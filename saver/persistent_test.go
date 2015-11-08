package saver_test

import (
	"github.com/th3osmith/greader/saver"
	"testing"
)

import _ "github.com/go-sql-driver/mysql"

func TestConnection(t *testing.T) {
	dbM, err := saver.NewDbManager("root:mypassa@tcp(127.0.0.1:3306)/greader")
	defer dbM.Close()

	if err == nil {
		t.Error("Bad Error Handling for bad credentials")
	}

	dbM, err = saver.NewDbManager("root:mypass@tcp(127.0.0.1:3306)/greader")
	defer dbM.Close()

	dbM.Ping()
	dbM.Close()

	if err != nil {
		t.Error(err)
		dbM.Close()
	}
}

type copain struct {
	Id     int
	Name   string
	Master *testSQL `type:"ManyToOne" object:"testSQL"`
}

type testSQL struct {
	Id      int
	Name    string
	Keya    string `type:"TEXT"`
	Keyb    int
	Keyx    int
	Copains []*copain `type:"OneToMany" object:"copain"`
}

func TestCreatePersistentSQL(t *testing.T) {

	obj := testSQL{1, "tata", "yoyo", 25, 42, nil}
	slave := copain{1, "toto", &obj}

	db, err := saver.NewDbManager("root:mypass@tcp(127.0.0.1:3306)/greader")
	defer db.Close()
	if err != nil {
		t.Error(err)
	}

	err = db.CreateTable(&obj)
	if err != nil {
		t.Error(err)
	}

	err = db.CreateTable(&slave)
	if err != nil {
		t.Error(err)
	}

	err = db.Save(&obj)
	if err != nil {
		t.Error(err)
	}

	err = db.Save(&slave)
	if err != nil {
		t.Error(err)
	}

	objA := new(testSQL)
	err = db.Retrieve(1, &objA)
	if err != nil {
		t.Error(err)
	}

	var copainA *copain
	err = db.Retrieve(1, &copainA)
	if err != nil {
		t.Error(err)
	}

	if objA.Name != "tata" {
		t.Error("Failure in object retrieval")
	}

	if copainA.Master.Name != "tata" {
		t.Error("Failure in object relation retrieval")
	}

	err = db.Retrieve(12, &objA)
	if err == nil {
		t.Error("Error not detected in object retrieval")
	}

	objA.Name = "Jeanne"
	err = db.Update(objA)
	if err != nil {
		t.Error(err)
	}

	objB := new(testSQL)
	err = db.Retrieve(1, &objB)
	objC := new(testSQL)
	err = db.Retrieve(1, &objC)

	if objB.Name != "Jeanne" {
		t.Error("Modification not saved in DB")
	}

	objB.Id = 4
	err = db.Update(objB)

	if err != nil && err.Error() != "Nothing updated" {
		t.Error(err)
	}

	objB.Id = 1

	err = db.Populate(objA)
	if err != nil {
		t.Error(err)
	}

	if len(objA.Copains) < 1 || objA.Copains[0].Name != "toto" {
		t.Error("Error retrieving relations")

	}

	err = db.Delete(objA)
	if err != nil {
		t.Error(err)
	}

	err = db.Delete(objA)
	if err != nil && err.Error() != "Object not in DB" {
		t.Error("Error badly handled in Delete")
	}

}
