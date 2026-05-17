package storage

func (s *Store) Migrate() error {
	stmts := []string{
		`create table if not exists vault (
			id integer primary key check (id = 1),
			password_hash text not null,
			created_at datetime not null default current_timestamp
		)`,
		`create table if not exists notes (
			id text primary key,
			path text not null unique,
			title text not null,
			nonce blob not null,
			ciphertext blob not null,
			created_at datetime not null default current_timestamp,
			updated_at datetime not null default current_timestamp
		)`,
		`create table if not exists folders (
			path text primary key,
			created_at datetime not null default current_timestamp
		)`,
		`create index if not exists idx_notes_updated on notes(updated_at desc)`,
		`create index if not exists idx_notes_path on notes(path)`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			return err
		}
	}
	return s.ensureNoteColumns()
}

func (s *Store) ensureNoteColumns() error {
	rows, err := s.db.Query(`pragma table_info(notes)`)
	if err != nil {
		return err
	}
	defer rows.Close()
	hasEyesOnly := false
	for rows.Next() {
		var cid int
		var name, typ string
		var notNull int
		var defaultValue any
		var pk int
		if err := rows.Scan(&cid, &name, &typ, &notNull, &defaultValue, &pk); err != nil {
			return err
		}
		if name == "eyes_only" {
			hasEyesOnly = true
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if !hasEyesOnly {
		_, err := s.db.Exec(`alter table notes add column eyes_only integer not null default 0`)
		return err
	}
	_, err = s.db.Exec(`update notes set eyes_only = case when lower(cast(eyes_only as text)) in ('1', 'true', 't', 'yes', 'y', 'on') then 1 else 0 end`)
	if err != nil {
		return err
	}
	return nil
}
