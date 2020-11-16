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

func TestBlobsOptionsFromViper(t *testing.T) {
	viper.Set("storage.blobs.foldername", "/very-secret/location")

	expected := BlobsOptions{
		Foldername: "/very-secret/location",
	}
	actual := BlobsOptionsFromViper()
	assert.Equal(t, expected, actual)
}

func TestBlobsTestSuite(t *testing.T) {
	suite.Run(t, new(BlobsTestSuite))
}

type BlobsTestSuite struct {
	baseFileystemTestSuite

	blobs Blobs
}

func (s *BlobsTestSuite) SetupTest() {
	s.baseFileystemTestSuite.SetupTest()

	blobs, err := NewBlobs(s.fs, s.idGen, BlobsOptions{Foldername: "/test/blobs"})
	s.Require().NoError(err)
	s.Require().NotNil(blobs)

	s.blobs = blobs
}

func (s *BlobsTestSuite) TestWrite() {
	const data = "TestWrite"

	s.idGen.On("GenerateID").Return(data, nil)

	id, size, err := s.blobs.Write(context.TODO(), strings.NewReader(data))
	s.Assert().NoError(err)
	s.Assert().Equal(data, id)
	s.Assert().EqualValues(len(data), size)

	s.assertFileContent("/test/blobs/TestWrite", data)
}

func (s *BlobsTestSuite) TestReaderInvalid() {
	_, err := s.blobs.Reader("")
	s.Assert().Error(err)
}

func (s *BlobsTestSuite) TestReaderNotFound() {
	_, err := s.blobs.Reader("not-existing")
	s.Assert().Error(err)
}

func (s *BlobsTestSuite) TestReaderOK() {
	const data = "TestReader-data"

	s.requireWrite("/test/blobs/TestReader-id", data)

	r, err := s.blobs.Reader("TestReader-id")
	s.Require().NoError(err)
	s.Require().NotNil(r)

	actual, err := ioutil.ReadAll(r)
	s.Assert().NoError(err)
	s.Assert().EqualValues(data, actual)
	s.Assert().NoError(r.Close())
}

func (s *BlobsTestSuite) TestOffsetReaderNotFound() {
	_, err := s.blobs.OffsetReader("not-existing", 10)
	s.Assert().Error(err)
}

func (s *BlobsTestSuite) TestOffsetReader() {
	const data = "TestOffsetReader-data"

	s.requireWrite("/test/blobs/TestOffsetReader-id", data)

	r, err := s.blobs.OffsetReader("TestOffsetReader-id", 4)
	s.Require().NoError(err)
	s.Require().NotNil(r)

	actual, err := ioutil.ReadAll(r)
	s.Assert().NoError(err)
	s.Assert().EqualValues(data[4:], actual)
}

func (s *BlobsTestSuite) TestDelete() {
	s.requireWrite("/test/blobs/TestDelete-id", "TestDelete")

	err := s.blobs.Delete(context.TODO(), "TestDelete-id")
	s.Require().NoError(err)
}
