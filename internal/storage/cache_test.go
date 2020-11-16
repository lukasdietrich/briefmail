// Copyright (C) 2019  Lukas Dietrich <lukas@lukasdietrich.com>
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package storage

import (
	"context"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

func TestCacheOptionsFromViper(t *testing.T) {
	viper.Set("storage.cache.foldername", "/super-secret/temporary")
	viper.Set("storage.cache.memorylimit", "123kb")

	expected := CacheOptions{
		Foldername:  "/super-secret/temporary",
		MemoryLimit: 123 * 1024,
	}
	actual := CacheOptionsFromViper()
	assert.Equal(t, expected, actual)
}

func TestCacheTestSuite(t *testing.T) {
	suite.Run(t, new(CacheTestSuite))
}

type CacheTestSuite struct {
	baseFileystemTestSuite

	cache Cache
}

func (s *CacheTestSuite) SetupTest() {
	s.baseFileystemTestSuite.SetupTest()

	cache, err := NewCache(s.fs, s.idGen, CacheOptions{Foldername: "/test/cache", MemoryLimit: 16})
	s.Require().NoError(err)
	s.Require().NotNil(cache)

	s.cache = cache
}

func (s *CacheTestSuite) TestInMemory() {
	const data = "TestInMemory"

	entry, err := s.cache.Write(context.TODO(), strings.NewReader(data))
	s.Require().NoError(err)
	s.Assert().IsType(memoryEntry{}, entry)
	s.assertMultipleReads(entry, data)

	s.Assert().NoError(entry.Release(context.TODO()))
}

func (s *CacheTestSuite) TestOnDisk() {
	const data = "TestOnDisk......"

	s.idGen.On("GenerateID").Return("TestOnDisk", nil)

	entry, err := s.cache.Write(context.TODO(), strings.NewReader(data))
	s.Require().NoError(err)
	s.Assert().IsType(fileEntry{}, entry)
	s.assertMultipleReads(entry, data)
	s.assertFileContent("/test/cache/TestOnDisk", data)

	s.Assert().NoError(entry.Release(context.TODO()))
}

func (s *CacheTestSuite) assertMultipleReads(entry CacheEntry, expectedContent string) {
	for i := 0; i < 3; i++ {
		r, err := entry.Reader()
		s.Require().NoError(err)
		s.Require().NotNil(r)

		actualContent, err := ioutil.ReadAll(r)
		s.Require().NoError(err)
		s.Require().NotNil(actualContent)

		s.Require().EqualValues(expectedContent, actualContent)
	}
}
