package saver_test

import (
	"database/sql"
	"github.com/th3osmith/greader/saver"
	"os"
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

var db *saver.DbManager

/*
 * Create Database and tables associated to the objects
 */
func TestCreateDatabase(t *testing.T) {

	obj := testSQL{}
	slave := copain{}

	var err error

	db, err = saver.NewDbManager("root:mypass@tcp(127.0.0.1:3306)/greader")
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
}

// DbManager adds objectSQL on the fly
func TestAddStruct(t *testing.T) {
	obj := testSQL{1, "tatatata", "yoyo", 25, 42, nil}

	dbM, err := saver.NewDbManager("root:mypass@tcp(127.0.0.1:3306)/greader")
	defer dbM.Close()

	err = dbM.Save(&obj)
	if err != nil {
		t.Error(err)
	}
}

/*
 * Save an object to DB
 * Get it from cache
 * Update it
 * Get it from DB
 * Delete it
 */
func TestCRUDObject(t *testing.T) {
	obj := testSQL{1, "tatatoto", "yoyo", 25, 42, nil}

	err := db.Save(&obj)
	if err != nil {
		t.Error(err)
	}

	ID := obj.Id

	// Retrieve from Cache
	var objA *testSQL
	err = db.Retrieve(ID, &objA)

	if objA.Name != "tatatoto" {
		t.Error("Error in Cache", err)
	}

	// Remove from Cache
	db.EjectFromCache(objA)

	// Retrieve from DB
	var objC *testSQL
	err = db.Retrieve(ID, &objC)

	if objC.Name != "tatatoto" {
		t.Error("Error in DB", err)
	}

	// Update object
	objC.Name = "totoro"

	err = db.Update(objC)
	if err != nil {
		t.Error(err)
	}

	db.EjectFromCache(objC)

	var objD *testSQL
	err = db.Retrieve(ID, &objD)

	if objD.Name != "totoro" {
		t.Error("Error in Update", err)
	}

	// Remove from DB
	err = db.Delete(&obj)
	if err != nil {
		t.Error(err)
	}

	// Try to get from anywhere
	var objB *testSQL
	err = db.Retrieve(ID, &objB)

	if err == nil || err != sql.ErrNoRows {
		t.Error("Error in Deletion")
	}

}

func TestRelation(t *testing.T) {

	obj := testSQL{1, "tata", "yoyo", 25, 42, nil}
	slave := copain{1, "toto", &obj}

	err := db.Save(&obj)
	if err != nil {
		t.Error(err)
	}

	err = db.Save(&slave)
	if err != nil {
		t.Error(err)
	}

	//idMaster := obj.Id
	//idSlave := slave.Id

	err = db.Populate(&obj)

	if len(obj.Copains) < 1 || obj.Copains[0].Name != "toto" {
		t.Error("Error in retrieving relation", err)
	}

	// Test deletion of the slave first
	db.Delete(&slave)

	if len(obj.Copains) > 0 || err != nil {
		t.Error("Error in Deletion propagation", err)
	}

}

func TestComplexObject(t *testing.T) {

	obj := testSQL{1, "tata", "yoyo", 25, 42, nil}
	slave := copain{1, "toto", &obj}

	err := db.Save(&obj)
	if err != nil {
		t.Error(err)
	}

	err = db.Save(&slave)
	if err != nil {
		t.Error(err)
	}

	ID := obj.Id
	IDA := slave.Id

	objA := new(testSQL)
	err = db.Retrieve(ID, &objA)
	if err != nil {
		t.Error(err)
	}

	var copainA *copain
	err = db.Retrieve(IDA, &copainA)
	if err != nil {
		t.Error(err)
	}

	if objA.Name != "tata" {
		t.Error("Failure in object retrieval")
	}

	if copainA.Master.Name != "tata" {
		t.Error("Failure in object relation retrieval")
	}

	err = db.Retrieve(ID+123, &objA)
	if err == nil {
		t.Error("Error not detected in object retrieval")
	}

	objA.Name = "Jeanne"
	err = db.Update(objA)
	if err != nil {
		t.Error(err)
	}

	objB := new(testSQL)
	err = db.Retrieve(ID, &objB)
	objC := new(testSQL)
	err = db.Retrieve(ID, &objC)

	if objB.Name != "Jeanne" {
		t.Error("Modification not saved in DB")
	}

	objB.Id = ID + 1
	err = db.Update(objB)

	if err != nil && err.Error() != "Nothing updated" {
		t.Error(err)
	}

	objB.Id = ID

	err = db.Populate(objA)
	if err != nil {
		t.Error(err)
	}

	if len(objA.Copains) < 1 || objA.Copains[0].Name != "toto" {
		t.Error("Error retrieving relations")

	}

	// Test Deletion of the master first
	err = db.Delete(objA)

	if copainA.Master != nil {
		t.Error("Error in deletion propagation", err)
	}

	err = db.Delete(copainA)
	if err != nil {
		t.Error(err)
	}

	err = db.Delete(objA)
	if err != nil && err.Error() != "Object not in DB" {
		t.Error("Error badly handled in Delete")
	}

}

func TestMain(m *testing.M) {

	// Setup DB

	os.Exit(m.Run())
	db.Close()

}
