package artifacts

import (
	db "gopkg.in/rethinkdb/rethinkdb-go.v6"
	"reflect"
	"testing"
	"time"
)

var specification, i = LoadSpecification()

func TestInitDbConnection(t *testing.T) {
	type args struct {
		s Specification
	}

	if i != nil {
		t.Errorf("failed %v", i)
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "can init a db connection", args: args{specification}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := InitDbConnection(tt.args.s)
			if (err != nil) != tt.wantErr {
				t.Errorf("InitDbConnection() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			server, err := got.Server()
			if err != nil {
				t.Errorf("error %v", err)
			}
			t.Log(server.Name, server.ID)
		})
	}
}

func TestRethinkStorage_Changes(t *testing.T) {
	type fields struct {
		s Specification
	}
	type args struct {
		status             []Status
		namespaceSubstring string
		packageIdSubstring string
		versions           chan Artifact
	}



	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{name: "can listen for changes", fields: fields{specification}, args: args{
			nil, "", "", make(chan Artifact),
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rs, err := InitDbConnection(tt.fields.s)
			if err != nil {
				t.Errorf("Failed to connect to db %v", err)
			}

			rs.Changes(tt.args.status, tt.args.namespaceSubstring, tt.args.packageIdSubstring, tt.args.versions)

			errprs := rs.Insert(Artifact{
				Repository: "sup",
				Namespace:  "dawg",
				Package:    "",
				Version:    "",
				Revision:   "",
				DomainName: "",
				Format:     "",
				Id:         "",
				Error:      nil,
				Status:     "",
				CreateTime: time.Time{},
			})
			if errprs[0] != nil {
				t.Fatal("Failed to insert a value to test changes function")
			}

			change := <-tt.args.versions

			if change.Repository != "sup" || change.Namespace != "dawg" {
				t.Errorf("Did not recieve expected aftifact out of channel.")
			}
		})
	}
}

func TestRethinkStorage_Insert(t *testing.T) {
	type fields struct {
		Session   *db.Session
		dbName    string
		tableName string
	}
	type args struct {
		artifacts []Artifact
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rs := &RethinkStorage{
				Session:   tt.fields.Session,
				dbName:    tt.fields.dbName,
				tableName: tt.fields.tableName,
			}
			rs.Server()
		})
	}
}

func TestRethinkStorage_List(t *testing.T) {
	type fields struct {
		Session   *db.Session
		dbName    string
		tableName string
	}
	type args struct {
		status             []Status
		namespaceSubstring string
		packageIdSubstring string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []Artifact
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rs := &RethinkStorage{
				Session:   tt.fields.Session,
				dbName:    tt.fields.dbName,
				tableName: tt.fields.tableName,
			}
			got, err := rs.List(tt.args.status, tt.args.namespaceSubstring, tt.args.packageIdSubstring)
			if (err != nil) != tt.wantErr {
				t.Errorf("List() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("List() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRethinkStorage_artifactsQuery(t *testing.T) {
	type fields struct {
		Session   *db.Session
		dbName    string
		tableName string
	}
	type args struct {
		status             []Status
		namespaceSubstring string
		packageIdSubstring string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   db.Term
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rs := &RethinkStorage{
				Session:   tt.fields.Session,
				dbName:    tt.fields.dbName,
				tableName: tt.fields.tableName,
			}
			if got := rs.artifactsQuery(tt.args.status, tt.args.namespaceSubstring, tt.args.packageIdSubstring); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("artifactsQuery() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRethinkStorage_ensureTableExists(t *testing.T) {
	type fields struct {
		Session   *db.Session
		dbName    string
		tableName string
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rs := &RethinkStorage{
				Session:   tt.fields.Session,
				dbName:    tt.fields.dbName,
				tableName: tt.fields.tableName,
			}
			if err := rs.ensureTableExists(); (err != nil) != tt.wantErr {
				t.Errorf("ensureTableExists() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRethinkStorage_table(t *testing.T) {
	type fields struct {
		Session   *db.Session
		dbName    string
		tableName string
	}
	tests := []struct {
		name   string
		fields fields
		want   db.Term
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rs := &RethinkStorage{
				Session:   tt.fields.Session,
				dbName:    tt.fields.dbName,
				tableName: tt.fields.tableName,
			}
			if got := rs.table(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("table() = %v, want %v", got, tt.want)
			}
		})
	}
}
