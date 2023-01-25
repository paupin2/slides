package data

type User struct {
	ID   string `json:"-"`
	Name string `json:"name"`
}

func (u User) Valid() bool {
	return u.ID != "" && CheckTitle(u.Name) == nil
}

func UnusedUserID() string {
	for {
		id := randomString(32)
		rows, err := runQuery(`
			select 1
			from users
			where id = ?
		`, id)
		if err == nil && rows.Next() {
			// id already used
			rows.Close()
			continue
		}
		return id
	}
}
