// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build !windows

package swapscraper

import (
	"context"
	"errors"
	"testing"

	"github.com/shirou/gopsutil/mem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/consumer/consumererror"
)

func TestScrape_Errors(t *testing.T) {
	type testCase struct {
		name              string
		virtualMemoryFunc func() (*mem.VirtualMemoryStat, error)
		swapMemoryFunc    func() (*mem.SwapMemoryStat, error)
		expectedError     string
		expectedErrCount  int
	}

	testCases := []testCase{
		{
			name:              "virtualMemoryError",
			virtualMemoryFunc: func() (*mem.VirtualMemoryStat, error) { return nil, errors.New("err1") },
			expectedError:     "err1",
			expectedErrCount:  swapUsageMetricsLen,
		},
		{
			name:             "swapMemoryError",
			swapMemoryFunc:   func() (*mem.SwapMemoryStat, error) { return nil, errors.New("err2") },
			expectedError:    "err2",
			expectedErrCount: pagingMetricsLen,
		},
		{
			name:              "multipleErrors",
			virtualMemoryFunc: func() (*mem.VirtualMemoryStat, error) { return nil, errors.New("err1") },
			swapMemoryFunc:    func() (*mem.SwapMemoryStat, error) { return nil, errors.New("err2") },
			expectedError:     "[err1; err2]",
			expectedErrCount:  swapUsageMetricsLen + pagingMetricsLen,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			scraper := newSwapScraper(context.Background(), &Config{})
			if test.virtualMemoryFunc != nil {
				scraper.virtualMemory = test.virtualMemoryFunc
			}
			if test.swapMemoryFunc != nil {
				scraper.swapMemory = test.swapMemoryFunc
			}

			err := scraper.Initialize(context.Background())
			require.NoError(t, err, "Failed to initialize swap scraper: %v", err)

			_, err = scraper.Scrape(context.Background())
			assert.EqualError(t, err, test.expectedError)

			isPartial := consumererror.IsPartialScrapeError(err)
			assert.True(t, isPartial)
			if isPartial {
				assert.Equal(t, test.expectedErrCount, err.(consumererror.PartialScrapeError).Failed)
			}
		})
	}
}
