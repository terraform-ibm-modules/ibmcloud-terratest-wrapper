package testhelper

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var ex = Exemptions{List: []string{"i_am_exempt", "i.am.also.exempt"}}

func TestIsExempt(t *testing.T) {

	assert.True(t, ex.IsExemptedResource("i_am_exempt"), "This should have been exempt")
	assert.True(t, ex.IsExemptedResource("i.am.also.exempt"), "This should have been exempt")

}

func TestIsNotExempt(t *testing.T) {

	assert.False(t, ex.IsExemptedResource("i_am_not_exempt"), "This should not have been exempt")

}
func TestEmptyExemptionList(t *testing.T) {
	empty := Exemptions{List: nil}

	assert.False(t, empty.IsExemptedResource("i_am_not_exempt"), "This should not have been exempt")
}
