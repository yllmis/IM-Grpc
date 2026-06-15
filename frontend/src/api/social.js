import { http } from './http'

// 好友
export function getFriends() {
  return http.get('/v1/social/friends')
}

export function getFriendOnline() {
  return http.get('/v1/social/friend/online')
}

export function getFriendPutInList() {
  return http.get('/v1/social/friend/putIns')
}

export function sendFriendRequest(userUid, reqMsg) {
  return http.post('/v1/social/friend/putIn', { user_uid: userUid, req_msg: reqMsg })
}

export function handleFriendRequest(friendReqId, handleResult) {
  return http.put('/v1/social/friend/putIn', { friend_req_id: friendReqId, handle_result: handleResult })
}

// 群组
export function getGroups() {
  return http.get('/v1/social/groups')
}

export function createGroup(name, icon) {
  return http.post('/v1/social/group', { name, icon })
}

export function getGroupUsers(groupId) {
  return http.get('/v1/social/group/users', { params: { group_id: groupId } })
}

export function getGroupOnline(groupId) {
  return http.get('/v1/social/group/online', { params: { group_id: groupId } })
}

export function getGroupPutInList(groupId) {
  return http.get('/v1/social/group/putIns', { params: { group_id: groupId } })
}

export function joinGroup(groupId, reqMsg) {
  return http.post('/v1/social/group/putIn', { group_id: groupId, req_msg: reqMsg })
}

export function handleGroupRequest(groupReqId, groupId, handleResult) {
  return http.put('/v1/social/group/putIn', { group_req_id: groupReqId, group_id: groupId, handle_result: handleResult })
}
