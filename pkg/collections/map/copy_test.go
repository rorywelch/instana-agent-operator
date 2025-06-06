/*
 * (c) Copyright IBM Corp. 2025
 */

package _map

import (
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/rand"
)

func TestCopierCopy(t *testing.T) {
	assertions := require.New(t)

	expected := map[string]string{
		rand.String(10): rand.String(10),
		rand.String(10): rand.String(10),
		rand.String(10): rand.String(10),
	}

	c := NewCopier(expected)

	actual := c.Copy()

	assertions.NotSame(&expected, &actual)
	assertions.Equal(expected, actual)
}
