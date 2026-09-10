package skills

import (
	"fmt"
	"os/user"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestName(t *testing.T) {
	user, err := user.Current()
	assert.NoError(t, err)
	fmt.Printf("User: %v\n", user.Name)
	fmt.Printf("User: %v\n", user.Username)
	fmt.Printf("User: %v\n", user.Uid)
	fmt.Printf("User: %v\n", user.Gid)
	//fmt.Printf("User: %v\n", user.GroupIds())
}
