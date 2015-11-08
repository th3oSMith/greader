package saver

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strconv"
)

type PersistentSQL struct {
	object        interface{}
	id            string
	structure     map[string]string
	fields        []string
	name          string
	relations     map[string]string
	keys          map[string]string
	saveStmt      *sql.Stmt
	getStmt       *sql.Stmt
	deleteStmt    *sql.Stmt
	updateStmt    *sql.Stmt
	relationsStmt map[string]*sql.Stmt
	populateStmt  map[string]*sql.Stmt
}

func (p *PersistentSQL) hasStmts() bool {
	return p.saveStmt != nil
}

var typeMapping = map[string]string{
	"int":    "INT",
	"string": "VARCHAR(255)",
}

func (m *DbManager) NewPersistentSQL(obj interface{}) (object *PersistentSQL, err error) {
	object, err = m.CreatePersistentSQL(obj)
	if err != nil {
		return nil, err
	}

	err = m.GenStmts(object)
	if err != nil {
		return nil, err
	}

	return object, nil

}

func (m *DbManager) CreatePersistentSQL(obj interface{}) (object *PersistentSQL, err error) {

	objType := reflect.TypeOf(obj).Elem()

	object = new(PersistentSQL)
	object.object = obj
	object.structure = make(map[string]string)
	object.relations = make(map[string]string)
	object.keys = make(map[string]string)
	object.fields = []string{}
	object.name = objType.Name()

	// Parsing of the fieds
	for i := 0; i < objType.NumField(); i++ {
		field := objType.Field(i)

		if objT := field.Tag.Get("type"); objT != "" {

			if objT == "OneToMany" {
				object.relations[field.Name] = field.Tag.Get("object")
			} else if objT == "ManyToOne" {
				object.keys[field.Name] = field.Tag.Get("object")
				object.structure[field.Name] = "INT(4)"
				object.fields = append(object.fields, field.Name)

			} else {
				object.structure[field.Name] = objT
				object.fields = append(object.fields, field.Name)
			}
		} else {
			object.structure[field.Name] = typeMapping[field.Type.Name()]
			object.fields = append(object.fields, field.Name)
		}

		if field.Tag.Get("id") != "" || (field.Name == "Id" && object.structure[field.Name] == "INT") {
			object.id = field.Name
		}

	}

	// Is there an id field ?
	if object.id == "" {
		return nil, errors.New("Object MUST have an id field")
	}

	m.objectsSQL[object.name] = object
	return
}

func (m *DbManager) GenStmts(object *PersistentSQL) error {

	// Generating the Statements
	objType := reflect.TypeOf(object.object).Elem()

	// Gathering the fields in order
	fields := make([]string, 0, objType.NumField())
	for i := 0; i < objType.NumField(); i++ {
		if objType.Field(i).Tag.Get("type") != "OneToMany" {
			fields = append(fields, objType.Field(i).Name)
		}
	}

	parameters := ""
	structure := ""
	structureWithId := ""
	updateFields := ""

	for _, field := range fields {
		if field == object.id {
			structureWithId += fmt.Sprintf(" %v, ", field)
			continue
		}
		structure += fmt.Sprintf(" %v, ", field)
		updateFields += fmt.Sprintf(" %v=?, ", field)
		structureWithId += fmt.Sprintf(" %v, ", field)
		parameters += "?, "
	}
	structure = structure[:len(structure)-2]
	structureWithId = structureWithId[:len(structureWithId)-2]
	parameters = parameters[:len(parameters)-2]
	updateFields = updateFields[:len(updateFields)-2]

	insert := fmt.Sprintf("INSERT INTO `%v` (%v) VALUES (%v);", object.name, structure, parameters)
	update := fmt.Sprintf("UPDATE `%v` SET %v WHERE %v = ? ;", object.name, updateFields, object.id)
	get := fmt.Sprintf("SELECT %v FROM `%v` WHERE %v = ?;", structureWithId, object.name, object.id)
	del := fmt.Sprintf("DELETE FROM %v WHERE %v = ?;", object.name, object.id)

	// Relations Statements
	object.relationsStmt = make(map[string]*sql.Stmt)
	object.populateStmt = make(map[string]*sql.Stmt)

	for relationField, target := range object.relations {
		targetSQL, err := m.GetPersistentSQLByName(target)
		if err != nil {
			return err
		}

		targetField := ""

		for field, key := range targetSQL.keys {
			if key == object.name {
				targetField = field
				break
			}
		}

		sql := fmt.Sprintf("UPDATE `%v` SET %v = NULL WHERE %v = ?;", target, targetField, targetField)
		stmt, err := m.db.Prepare(sql)
		if err != nil {
			return err
		}
		object.relationsStmt[relationField] = stmt

		sql = fmt.Sprintf("SELECT %v FROM `%v` WHERE %v = ?;", targetSQL.id, target, targetField)

		stmt, err = m.db.Prepare(sql)
		if err != nil {
			return err
		}
		object.populateStmt[relationField] = stmt

	}

	var err error

	object.saveStmt, err = m.db.Prepare(insert)
	object.getStmt, err = m.db.Prepare(get)
	object.deleteStmt, err = m.db.Prepare(del)
	object.updateStmt, err = m.db.Prepare(update)

	//TODO Improve error

	if err != nil {
		return err
	}

	return nil
}

type DbManager struct {
	db         *sql.DB
	objectsSQL map[string]*PersistentSQL
	store      map[string]interface{}
}

func (m *DbManager) CreateTable(objRaw interface{}) error {

	// Prepare the SQL Requests
	// NB: Impossible to prepare SQL statement on DROP and CREATE TABLE

	obj, err := m.CreatePersistentSQL(objRaw)
	if err != nil {
		return err
	}

	dropTable := fmt.Sprintf("DROP TABLE IF EXISTS `%v`;", obj.name)
	disableCheck := "SET FOREIGN_KEY_CHECKS=0;"
	enableCheck := "SET FOREIGN_KEY_CHECKS=1;"

	createTable := fmt.Sprintf("CREATE TABLE `%v` (", obj.name)

	for field, fieldType := range obj.structure {
		createTable += fmt.Sprintf(" %v %v , ", field, fieldType)
		if field == obj.id {
			createTable = createTable[:len(createTable)-2]
			createTable += " AUTO_INCREMENT, "
		}
	}

	for field, target := range obj.keys {
		objSQL, err := m.GetPersistentSQLByName(target)
		if err != nil {
			return err
		}
		targetId := objSQL.id
		createTable += fmt.Sprintf("FOREIGN KEY (%v) REFERENCES %v(%v), ", field, target, targetId)
	}

	createTable += fmt.Sprintf("PRIMARY KEY (%v)", obj.id)

	createTable += ") ENGINE=INNODB;"

	_, err = m.db.Exec(disableCheck)
	if err != nil {
		return err
	}

	_, err = m.db.Exec(dropTable)
	if err != nil {
		return err
	}

	_, err = m.db.Exec(createTable)
	if err != nil {
		return err
	}

	_, err = m.db.Exec(enableCheck)
	if err != nil {
		return err
	}

	return nil

}

func (m *DbManager) GetPersistentSQLByName(name string) (objSQL *PersistentSQL, err error) {

	objSQL, ok := m.objectsSQL[name]

	if !ok {
		return nil, errors.New(fmt.Sprintf("Object %v  not referenced", name))

	}

	return objSQL, nil

}

func (m *DbManager) GetPersistentSQL(obj interface{}) (objSQL *PersistentSQL, err error) {

	name := reflect.TypeOf(obj).Elem().Name()
	objSQL, ok := m.objectsSQL[name]

	if !ok {
		objSQL, err = m.NewPersistentSQL(obj)
		if err != nil {
			return nil, err
		}

	}

	if !objSQL.hasStmts() {
		err = m.GenStmts(objSQL)
		if err != nil {
			return nil, err
		}
	}

	return objSQL, nil

}

func (m *DbManager) Save(obj interface{}) error {

	objValue := reflect.ValueOf(obj).Elem()
	objectSQL, err := m.GetPersistentSQL(obj)

	values, err := m.getObjValues(obj)

	if err != nil {
		return err
	}

	results, err := objectSQL.saveStmt.Exec(values...)
	if err != nil {
		return err
	}

	id, err := results.LastInsertId()
	if err != nil {
		return err
	}

	field := objValue.FieldByName(objectSQL.id)
	field.SetInt(id)

	return err

}

func (m *DbManager) Update(obj interface{}) error {

	objValue := reflect.ValueOf(obj).Elem()
	objectSQL, err := m.GetPersistentSQL(obj)

	values, err := m.getObjValues(obj)

	// We need to add the ID
	field := objValue.FieldByName(objectSQL.id)
	values = append(values, strconv.FormatInt(field.Int(), 10))

	if err != nil {
		return err
	}

	result, err := objectSQL.updateStmt.Exec(values...)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if affected < 1 {
		return errors.New("Nothing updated")
	}

	return err

}

func (m *DbManager) getObjValues(obj interface{}) ([]interface{}, error) {

	objType := reflect.TypeOf(obj).Elem()
	objValue := reflect.ValueOf(obj).Elem()

	objectSQL, err := m.GetPersistentSQL(obj)
	if err != nil {
		return nil, err
	}

	values := make([]interface{}, 0, objType.NumField())

	for i := 0; i < objType.NumField(); i++ {
		field := objValue.Field(i)
		fieldName := objType.Field(i).Name

		if fieldName == objectSQL.id || objType.Field(i).Tag.Get("type") == "OneToMany" {
			continue
		}

		if objType.Field(i).Tag.Get("type") == "ManyToOne" {
			targetSQL, err := m.GetPersistentSQLByName(objectSQL.keys[fieldName])

			if err != nil {
				return nil, err
			}

			targetField := field.Elem().FieldByName(targetSQL.id)
			values = append(values, strconv.FormatInt(targetField.Int(), 10))
			continue

		}

		switch field.Kind() {
		case reflect.String:
			values = append(values, field.String())
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			values = append(values, strconv.FormatInt(field.Int(), 10))
		default:
			return nil, errors.New(fmt.Sprintf("Type not handled: %v", objValue.Kind()))

		}

	}

	return values, nil

}

func (m *DbManager) Delete(obj interface{}) error {
	objectSQL, err := m.GetPersistentSQL(obj)
	if err != nil {
		return err
	}

	objValue := reflect.ValueOf(obj).Elem()
	field := objValue.FieldByName(objectSQL.id)
	ID := field.Int()

	// Take care of dependencies
	for field := range objectSQL.relations {
		_, err := objectSQL.relationsStmt[field].Exec(ID)
		if err != nil {
			return err
		}
	}

	result, err := objectSQL.deleteStmt.Exec(ID)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if affected < 1 {
		return errors.New("Object not in DB")
	}

	return nil
}

func (m *DbManager) Populate(obj interface{}) error {
	objectSQL, err := m.GetPersistentSQL(obj)
	if err != nil {
		return err
	}

	objValue := reflect.ValueOf(obj).Elem()
	field := objValue.FieldByName(objectSQL.id)
	ID := field.Int()

	// Loading all relations
	for field := range objectSQL.relations {
		rows, err := objectSQL.populateStmt[field].Query(ID)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var childId int
			if err := rows.Scan(&childId); err != nil {
				return err
			}

			fieldType, _ := reflect.TypeOf(obj).Elem().FieldByName(field)
			tmp := reflect.New(fieldType.Type.Elem())
			ff := tmp.Elem().Interface()

			err := m.Retrieve(childId, &ff)
			if err != nil {
				return err
			}
			objValue.FieldByName(field).Set(reflect.Append(objValue.FieldByName(field), reflect.ValueOf(ff)))
		}
		if err := rows.Err(); err != nil {
			return err
		}
	}

	return nil

}

func (m *DbManager) Retrieve(ID int, dst interface{}) error {

	obj := reflect.Indirect(reflect.ValueOf(dst)).Interface()
	fmt.Println(reflect.TypeOf(obj))

	objectSQL, err := m.GetPersistentSQL(obj)
	if err != nil {
		return err
	}

	var values []interface{}

	// Try and get it from the store
	key := fmt.Sprintf("%v%v", objectSQL.name, ID)
	fmt.Println("Try to retrieve", key)

	if data, ok := m.store[key]; ok {
		reflect.ValueOf(dst).Elem().Set(data.(reflect.Value))
		return nil
	}

	objType := reflect.TypeOf(obj).Elem()
	point := reflect.New(objType)
	objValue := point.Elem()

	relationContainer := make(map[string]*int)

	for _, field := range objectSQL.fields {
		if _, ok := objectSQL.keys[field]; ok {
			fieldType, _ := reflect.TypeOf(obj).Elem().FieldByName(field)
			objValue.FieldByName(field).Set(reflect.New(fieldType.Type.Elem()))
			relationContainer[field] = new(int)
			values = append(values, relationContainer[field])

		} else {
			values = append(values, objValue.FieldByName(field).Addr().Interface())
		}
	}

	err = objectSQL.getStmt.QueryRow(ID).Scan(values...)
	if err != nil {
		return err
	}

	// Now we place value back
	for field, id := range relationContainer {
		fieldType, _ := reflect.TypeOf(obj).Elem().FieldByName(field)
		tmp := reflect.New(fieldType.Type.Elem())
		ff := tmp.Elem().Addr().Interface()

		err := m.Retrieve(*id, &ff)
		if err != nil {
			return err
		}
		objValue.FieldByName(field).Set(reflect.ValueOf(ff))
	}

	m.store[key] = objValue.Addr()
	reflect.ValueOf(dst).Elem().Set(objValue.Addr())

	return nil
}

func (m *DbManager) Close() {
	m.db.Close()
}

func (m *DbManager) Ping() error {
	return m.db.Ping()
}

func NewDbManager(credentials string) (dbManager *DbManager, err error) {

	dbManager = new(DbManager)

	dbManager.db, err = sql.Open("mysql", credentials)

	if err != nil {
		return
	}

	err = dbManager.db.Ping()
	if err != nil {
		return
	}

	dbManager.objectsSQL = make(map[string]*PersistentSQL)
	dbManager.store = make(map[string]interface{})

	return

}
