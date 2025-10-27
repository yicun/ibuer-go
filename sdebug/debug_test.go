package sdebug

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

type User struct {
	SDebugInfo
	ID   int    `json:"id"`
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func TestDefaultDebugInfo(t *testing.T) {
	user := &User{
		ID:   1,
		Name: "test",
		Age:  10,
	}

	err := user.SetDebugInfo(map[string]any{
		"first": "init",
	})
	require.NoError(t, err)

	err = user.AddDebugInfo("second", "test")
	require.NoError(t, err)

	err = user.AddDebugInfo2("third", "k1", "v1")
	require.NoError(t, err)

	jsonBytes, err := json.Marshal(user)
	require.NoError(t, err)
	t.Log(string(jsonBytes))

	user2 := &User{}
	err = json.Unmarshal(jsonBytes, user2)
	require.NoError(t, err)

	debugMap, err := user2.GetDebugInfoMap()
	require.NoError(t, err)
	t.Log(debugMap)

	jsonBytes2, err := json.Marshal(user2)
	require.NoError(t, err)
	t.Log(string(jsonBytes2))
}
