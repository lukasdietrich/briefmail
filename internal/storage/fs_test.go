// Copyright (C) 2020  Lukas Dietrich <lukas@lukasdietrich.com>
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
	"io"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/lukasdietrich/briefmail/internal/crypto"
)

func TestNewFilesystem(t *testing.T) {
	fs := NewFilesystem()

	assert.NotNil(t, fs)
	assert.Implements(t, (*afero.Fs)(nil), fs)
}

type baseFileystemTestSuite struct {
	suite.Suite

	fs    afero.Fs
	idGen *crypto.MockIDGenerator
}

func (s *baseFileystemTestSuite) SetupTest() {
	s.fs = afero.NewMemMapFs()
	s.idGen = new(crypto.MockIDGenerator)
}

func (s *baseFileystemTestSuite) TeardownTest() {
	mock.AssertExpectationsForObjects(s.T(), s.idGen)
}

func (s *baseFileystemTestSuite) requireWrite(filename string, content string) {
	f, err := s.fs.Create(filename)
	s.Require().NoError(err)

	defer f.Close()
	_, err = io.Copy(f, strings.NewReader(content))
	s.Require().NoError(err)
}

func (s *baseFileystemTestSuite) assertFileContent(filename string, expectedContent string) {
	f, err := s.fs.Open(filename)
	s.Require().NoError(err)

	defer f.Close()
	actualContent, err := ioutil.ReadAll(f)
	s.Require().NoError(err)
	s.Assert().EqualValues(expectedContent, actualContent)
}
