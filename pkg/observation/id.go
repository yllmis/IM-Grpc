package observation

import "go.mongodb.org/mongo-driver/v2/bson"

func newObjectIDHex() string {
	return bson.NewObjectID().Hex()
}

// ParseObjectIDHex 解析 24hex ObjectID；失败返回 false。
func ParseObjectIDHex(s string) (string, bool) {
	oid, err := bson.ObjectIDFromHex(s)
	if err != nil {
		return "", false
	}
	return oid.Hex(), true
}
