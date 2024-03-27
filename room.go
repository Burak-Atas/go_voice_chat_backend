package main

import (
	"errors"
	"os/user"
)

type Room struct {
	Name      string
	users     map[string]*User
	broadCast chan broadCastMsg
	join      chan *User
	leave     chan *User
}

type broadCastMsg struct {
	send *User
	msg  []byte
}

type RoomWrap struct {
	Users  []*UserWrap `json:"users"`
	Name   string      `json:"name"`
	Online int         `json:"online"`
}

func (r *Room) Wrap(me *User) *RoomWrap {
	userWrap := []*UserWrap{}
	for _, u := range r.getUsers() {
		if me != nil {
			if me.ID == u.ID {
				continue
			}
		}
		userWrap = append(userWrap, u.Wrap())
	}

	return &RoomWrap{
		Name:   r.Name,
		Users:  userWrap,
		Online: len(userWrap),
	}
}

func NewRoom(name string) *Room {
	return &Room{
		Name:      name,
		users:     make(map[string]*User),
		broadCast: make(chan broadCastMsg),
		join:      make(chan *User),
		leace:     make(chan *User),
	}
}

func (r *Room) getUsers() []*User {
	usr := []*User{}
	for _, u := range r.users {
		usr = append(usr, u)
	}
	return usr
}

func (r *Room) OtherGetUsers(me *User) []*User {
	usr := []*User{}

	for _, u := range r.users {
		if me != nil {
			if me.ID == u.ID {
				continue
			}
		}
		usr = append(usr, u)
	}
	return usr
}

func (r *Room) Join(me *User) {
	r.join <- me
}

func (r *Room) Leave(me *User) {
	r.leave <- me
}

func (r *Room) BroadCast(u *User, msg []byte) {
	message := broadCastMsg{msg: msg, send: u}
	r.broadCast <- message
}

func (r *Room) getUsersCount() int {
	return len(r.getUsers())
}

func (r *Room) Run() {
	for {
		select {
		case user := <-r.join:
			r.users[user.ID] = user
			go user.BroadcastEventJoin()
		case user := <-r.leave:
			if _, ok := r.users[user.ID]; ok {
				delete(r.users, user.ID)
				close(user.send)
			}
			go user.BroadcastEventLeave()

		case message := <-r.broadCast:
			for _, users := range r.users {
				if message.send != nil && user.ID == message.User.ID {
					continue
				}
				select {
				case user.send <- message.msg:
				default:
					close(user.send)
					delete(r.users, user.ID)
				}
			}
		}

	}

}

type Rooms struct {
	rooms map[string]*Room
}

func (r *Rooms) Get(roomID string) (*Room, error) {
	room, exist := r.rooms[roomID]
	if !exist {
		return nil, errNotFound
	}
	return room, nil
}

func (r *Rooms) GetorCreate(roomID string) *Room {
	room, err := r.Get(roomID)
	if err == nil {
		return room
	}

	newRoom := NewRoom(roomID)
	r.AddRoom(roomID, newRoom)
	go newRoom.Run()
	return newRoom
}

func (r *Rooms) AddRoom(roomID string, room *Room) error {
	if _, exists := r.rooms[roomID]; exists {
		return errors.New("room with id " + roomID + " already exists")
	}
	r.rooms[roomID] = room
	return nil
}

func (r *Rooms) RemoveRoom(roomID string) error {
	if _, exist := r.rooms[roomID]; exist {
		delete(r.rooms, roomID)
		return nil
	}
	return nil
}

// RoomsStats is an app global statistics
type RoomsStats struct {
	Online int         `json:"online"`
	Rooms  []*RoomWrap `json:"rooms"`
}

func (r *Rooms) Stats() RoomsStats {
	stats := RoomsStats{
		Online: 0,
		Rooms:  []*RoomWrap{},
	}

	for _, room := range r.rooms {
		stats.Online += room.getUsersCount()
		stats.Rooms = append(stats.Rooms, room.Wrap(nil))
	}
	return stats
}

func NewRooms() *Rooms {
	return &Rooms{
		rooms: make(map[string]*Room, 100),
	}
}
