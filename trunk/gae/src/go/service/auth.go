package service

import (
	"appengine"
	"appengine/capability"
	"event"
	"rand"
	//"bytes"
	//"fmt"
)

const ANONYMOUSE_NAME = "anonymouse"
const SEED = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz~!@#$%^&*()-+<>"

func generateRandomString(n int) string {
	s := ""
	for i := 0; i < n; i++ {
		index := rand.Intn(len(SEED))
		s += SEED[index : index+1]
	}
	return s
}

func generateAuthToken(ctx appengine.Context) string {
	token := generateRandomString(10)
	if nil != GetUserWithToken(ctx, token) {
		return generateAuthToken(ctx)
	}
	return token
}

func Auth(ctx appengine.Context, ev *event.AuthRequestEvent) event.Event {
	user := GetUserWithName(ctx, ev.User)
	res := new(event.AuthResponseEvent)
	res.Appid = ev.Appid
	if nil != user {
		res.Token = user.AuthToken
	} else {
		res.Error = "Invalid user/passwd."
	}
	return res
}

func assertRootAuth(user *event.User)string{
    if user.Email != "root"{
        return "You have no authorization for this operation!"
    }
    return ""
}

func HandlerUserEvent(ctx appengine.Context, ev *event.UserOperationEvent) event.Event {
	var res string
	resev := new(event.AdminResponseEvent)
	tags := ((ev.GetAttachement().([]interface{}))[0]).(*event.EventHeaderTags)
	opruser := GetUserWithToken(ctx, tags.Token)
	if nil == opruser{
	  resev.ErrorCause="Invalid user token" 
	  return resev
	}
	switch ev.Operation {
	case event.OPERATION_ADD:
	   res = createUser(ctx, opruser, ev.User.Email, ev.User.Group)
	case event.OPERATION_DELETE:
	   res = deleteUser(ctx, opruser, ev.User.Email)
	case event.OPERATION_MODIFY:
	    res = modifyUser(ctx, opruser, ev.User.Email, ev.User.Passwd)
	}
	resev.Response="Success"
	resev.ErrorCause=res
	return  resev
}

func HandlerUserListEvent(ctx appengine.Context, ev *event.ListUserRequestEvent) event.Event {
    var res string
	tags := ((ev.GetAttachement().([]interface{}))[0]).(*event.EventHeaderTags)
	opruser := GetUserWithToken(ctx, tags.Token)
	if nil == opruser{
	  resev := new(event.AdminResponseEvent)
	  resev.ErrorCause="Invalid user token" 
	  return resev
	}
	res = assertRootAuth(opruser)
	if len(res) > 0{
	   resev := new(event.AdminResponseEvent)
	   resev.ErrorCause=res
	   return resev
	}
	users := GetAllUsers(ctx)
	resev := new(event.ListUserResponseEvent)
	resev.Users = users
	return  resev
}

func HandlerGroupEvent(ctx appengine.Context, ev *event.GroupOperationEvent) event.Event {
    var res string
	resev := new(event.AdminResponseEvent)
	tags := ((ev.GetAttachement().([]interface{}))[0]).(*event.EventHeaderTags)
	opruser := GetUserWithToken(ctx, tags.Token)
	if nil == opruser{
	  resev.ErrorCause="Invalid user token" 
	  return resev
	}
	switch ev.Operation {
	case event.OPERATION_ADD:
	   res = createGroup(ctx,opruser, ev.Group.Name)
	case event.OPERATION_DELETE:
	   res = deleteGroup(ctx,opruser, ev.Group.Name)
	}
	resev.Response="Success"
	resev.ErrorCause=res
	return  resev
}

func HandlerGroupListEvent(ctx appengine.Context, ev *event.ListGroupRequestEvent) event.Event {
    var res string
	tags := ((ev.GetAttachement().([]interface{}))[0]).(*event.EventHeaderTags)
	opruser := GetUserWithToken(ctx, tags.Token)
	if nil == opruser{
	  resev := new(event.AdminResponseEvent)
	  resev.ErrorCause="Invalid user token" 
	  return resev
	}
	res = assertRootAuth(opruser)
	if len(res) > 0{
	   resev := new(event.AdminResponseEvent)
	   resev.ErrorCause=res
	   return resev
	}
	groups := GetAllGroups(ctx)
	resev := new(event.ListGroupResponseEvent)
	resev.Groups = groups
	return  resev
}

func HandlerBalcklistEvent(ctx appengine.Context, ev *event.BlackListOperationEvent) event.Event {
    var res string
	tags := ((ev.GetAttachement().([]interface{}))[0]).(*event.EventHeaderTags)
	opruser := GetUserWithToken(ctx, tags.Token)
	if nil == opruser{
	  resev := new(event.AdminResponseEvent)
	  resev.ErrorCause="Invalid user token" 
	  return resev
	}
	res = assertRootAuth(opruser)
	if len(res) > 0{
	   resev := new(event.AdminResponseEvent)
	   resev.ErrorCause=res
	   return resev
	}
	user := GetUserWithName(ctx, ev.User)
	var group *event.Group
	var blacklist map[string]string
	if nil != user{
	   blacklist = user.BlackList
	}else{
	   group = GetGroup(ctx, ev.Group)
	   if nil != group{
	      blacklist = group.BlackList
	   }else{
	       resev := new(event.AdminResponseEvent)
	      resev.ErrorCause="Invalid user/group for blaclist operation"
	      return resev
	   }
	}
	switch ev.Operation{
	   case event.BLACKLIST_ADD:
	      blacklist[ev.Host] = ev.Host
	   case event.BLACKLIST_DELETE:
	      blacklist[ev.Host] = "", false
	}
	if nil != user{
	   SaveUser(ctx, user)
	}else{
	   SaveGroup(ctx, group)
	}
	resev := new(event.AdminResponseEvent)
	resev.Response="Success"
	return  resev
}

func CheckDefaultAccount(ctx appengine.Context) {
	if !capability.Enabled(ctx, "datastore_v3", "*") || !capability.Enabled(ctx, "memcache", "*") {
		return
	}
	CreateGroupIfNotExist(ctx, "root")
	CreateUserIfNotExist(ctx, "root", "root")
	CreateGroupIfNotExist(ctx, "public")
	CreateGroupIfNotExist(ctx, "anonymouse")
	CreateUserIfNotExist(ctx, "anonymouse", "anonymouse")
}

func deleteUser(ctx appengine.Context, opr *event.User, email string) string{
    var res string
	res = assertRootAuth(opr)
	if len(res) == 0{
	  user := GetUserWithName(ctx, email)
	  if nil == user{
	     return "User not found!"
	  }
	  DeleteUser(ctx,user)
	}
	return res
}

func modifyUser(ctx appengine.Context, opr *event.User, email string, passwd string) string{
    var res string
	res = assertRootAuth(opr)
	if len(res) == 0{
	  user := GetUserWithName(ctx, email)
	  if nil == user{
	     return "User not found!"
	  }
	  if len(passwd) ==0{
	     return "New password can't be empty!"
	  }
	  user.Passwd = passwd
	  SaveUser(ctx,user)
	}
	return res
}

func createUser(ctx appengine.Context, opr *event.User, email string, groupName string) string{
    var res string
	res = assertRootAuth(opr)
	if len(res) == 0{
	  group := GetGroup(ctx, groupName)    
	  if nil == group{
	     return "Group not found!"
	  }
	  user := GetUserWithName(ctx, email)
	  if nil != user{
	     return "User already exist!"
	  }
	  user = new(event.User)
	  user.Email = email
	  user.Group = groupName
	  user.Passwd = generateRandomString(8)
	  user.AuthToken = generateAuthToken(ctx)
	  SaveUser(ctx, user)
	}
	return res
}

func createGroup(ctx appengine.Context, opr *event.User, groupName string)string{
     var res string
	 res = assertRootAuth(opr)
	if len(res) == 0{
	  group := GetGroup(ctx, groupName)    
	  if nil != group{
	     return "Group already exist!"
	  }
	  group = new(event.Group)
	  group.Name = groupName
	  SaveGroup(ctx, group)
	}
	return res
}

func deleteGroup(ctx appengine.Context, opr *event.User,groupName string)string{
     var res string
	res = assertRootAuth(opr)
	if len(res) == 0{
	  group := GetGroup(ctx, groupName)
	  if nil == group{
	     return "Group not found!"
	  }
	  DeleteGroup(ctx,group)
	}
	return res
}

func CreateUserIfNotExist(ctx appengine.Context, email string, groupName string) {
	user := GetUserWithName(ctx, email)
	if nil == user {
		user = new(event.User)
		user.Email = email
		user.Group = groupName
		if email == ANONYMOUSE_NAME {
			user.Passwd = ANONYMOUSE_NAME
		} else {
			user.Passwd = generateRandomString(10)
		}
		user.AuthToken = generateAuthToken(ctx)
		SaveUser(ctx, user)
	} else {
		if len(user.AuthToken) == 0 {
			user.AuthToken = generateAuthToken(ctx)
			SaveUser(ctx, user)
		}
	}
}

func CreateGroupIfNotExist(ctx appengine.Context, name string) {
	group := GetGroup(ctx, name)
	if nil == group {
		group = new(event.Group)
		group.Name = name
		SaveGroup(ctx, group)
	}
}
