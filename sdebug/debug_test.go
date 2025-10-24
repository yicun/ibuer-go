package sdebug

import (
	"encoding/json"
	"testing"
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
	user.SetDebugInfo(map[string]any{
		"first": "init",
	})
	user.AddDebugInfo("second", "test")
	user.AddDebugInfo2("third", "k1", "v1")
	jsonBytes, _ := json.Marshal(user)
	t.Log(string(jsonBytes))

	user2 := &User{}
	_ = json.Unmarshal(jsonBytes, user2)
	t.Log(user2.GetDebugInfoMap())
	jsonBytes2, _ := json.Marshal(user2)
	t.Log(string(jsonBytes2))
}
