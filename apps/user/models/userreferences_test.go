package models

import (
	"strings"
	"testing"
)

func TestReferenceQueryIsBoundedAndMinimal(t *testing.T) {
	query, args, err := referenceQuery("`users`", UserReferenceFilter{Nickname: "x%_=' OR 1=1", Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(query, "limit ?") || !strings.Contains(query, "order by `id`") || args[1] != int64(21) {
		t.Fatalf("%s %+v", query, args)
	}
	if strings.Contains(query, "OR 1=1") || args[0] != "%x=%=_==' OR 1==1%" {
		t.Fatalf("parameter escaping: %s %+v", query, args)
	}
	projection := strings.Split(query, " from ")[0]
	for _, field := range []string{"password", "phone", "avatar", "*"} {
		if strings.Contains(projection, field) {
			t.Fatal("sensitive projection", projection)
		}
	}
}
func TestReferenceQueryRejectsUnboundedOrAmbiguousInput(t *testing.T) {
	for _, filter := range []UserReferenceFilter{{UserID: "u1"}, {UserID: "u1", Limit: 201}, {Limit: 10}, {UserID: "u1", Phone: "123", Limit: 10}} {
		if _, _, err := referenceQuery("`users`", filter); err == nil {
			t.Fatalf("accepted %+v", filter)
		}
	}
}
