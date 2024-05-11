package redicoo

import (
	"os"
	"testing"
	"fmt"
	"strconv"
	"encoding/json"
	// "time"
	"errors"
)


type TestStoreValue struct {
	TestId string
	TestValue string
}


func (t TestStoreValue) Encode() ([]byte , error) {
	
	return json.Marshal(t)

}

func (t *TestStoreValue) Decode(input []byte) error {
	
	err := json.Unmarshal(input , t)

	if err != nil {
		return err
	}

	return nil
}

func getConninfo() ConnInfo {

	redisAddr , ok := os.LookupEnv("REDIS_ADDR")

	if !ok {
		panic("env REDIS_ADDR is not set")
	}

	redisDBStr , ok := os.LookupEnv("REDIS_DB")

	if !ok {
		panic("env REDIS_DB is not set")
	}


	redisDB , err := strconv.Atoi(redisDBStr)

	if err != nil {
		panic("env REDIS_DB is not valid")
	}

	var life int64 = 1800


	return ConnInfo{
		Addr : redisAddr,
		DB : redisDB,
		Lifetime : &life,
	}
}



func TestNewSessionId(t *testing.T) {

	rc := NewClient(getConninfo())

	ss , err := rc.NewSessionId()

	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(*ss)

	res ,err := rc.IsExists(*ss)

	if err != nil {
		t.Fatal(err)
	}

	if !res {
		t.Fatal(err)
	} 
}


func TestIsExists(t *testing.T) {

	rc := NewClient(getConninfo())

	ss , err := rc.NewSessionId()

	if err != nil {
		t.Fatal(err)
	}

	exists , err := rc.IsExists(*ss)

	if err != nil {
		t.Fatal(err)
	}

	if !exists {
		t.Fatal("存在しているはずのキーの存在が確認できません。")
	}


	exists , err = rc.IsExists(fmt.Sprintf("%sa",*ss))

	if err == nil {
		t.Fatal("存在しないはずのキーがエラーになっていません。")
	}

	if !errors.Is(err , ErrorKeyNotFound) {
		t.Fatal(err)
	}

	if exists {
		t.Fatal("存在しないはずのキーがエラーになっていません。")
	}
}



func TestGet(t *testing.T) {

	rc := NewClient(getConninfo())

	ss , err := rc.NewSessionId()

	if err != nil {
		t.Fatal(err)
	}


	testsuffix := "testsuffix"

	// key only
	_ , err = rc.Get(*ss , testsuffix)

	if err == nil {
		t.Fatal("存在しないはずのSuffixがエラーになっていません。")
	}

	if !errors.Is(err , ErrorSuffixNotFound) {
		t.Fatal(err)
	}



	test := TestStoreValue{
		TestId : "123" , 
		TestValue : "XXXXAAAA",
	}

	b , err := test.Encode()

	if err != nil {
		t.Fatal(err)
	}

	err = rc.Set(*ss , testsuffix , b)

	if err != nil {
		t.Fatal(err)
	}


	val , err := rc.Get(*ss , testsuffix)

	if err != nil {
		t.Fatal(err)
	}

	out := TestStoreValue{}
	err = out.Decode(val.([]byte))

	if err != nil {
		t.Fatal(err)
	}

	if out != test {
		t.Fatal("一致していません。")
	}

}



func TestSet(t *testing.T) {

	type Input struct {
		Suffix string
		TestVal TestStoreValue
	}

	rc := NewClient(getConninfo())

	ss , err := rc.NewSessionId()

	if err != nil {
		t.Fatal(err)
	}


	tests := []Input{
		Input{"testsuffix1",TestStoreValue{TestId : "123" , TestValue : "XXXXAAAA"}},
		Input{"testsuffix2",TestStoreValue{TestId : "234" , TestValue : "XXXXBBBB"}},
	}


	for _ , v := range(tests) {

		b , err := v.TestVal.Encode()

		if err != nil {
			t.Fatal(err)
		}

		err = rc.Set(*ss , v.Suffix , b)

		if err != nil {
			t.Fatal(err)
		}

	}


	for _ , v := range(tests) {

		val , err := rc.Get(*ss , v.Suffix)

		if err != nil {
			t.Fatal(err)
		}

		out := TestStoreValue{}
		err = out.Decode(val.([]byte))

		if err != nil {
			t.Fatal(err)
		}

		if out != v.TestVal  {
			t.Fatal("一致していません。")
		}
	}

// Update and Add

	// update index 0
	tests[0] = Input{"testsuffix1" , TestStoreValue{TestId : "999" , TestValue : "YYYYYAAAA"}}
	// add data. index = 2
	tests = append(tests , Input{"testsuffix3" , TestStoreValue{TestId : "345" , TestValue : "XXXXXCCCC"}})

	idxs := []int{0,2}

	for _ , k := range(idxs) {

		b , err := tests[k].TestVal.Encode()

		if err != nil {
			t.Fatal(err)
		}
	
		err = rc.Set(*ss , tests[k].Suffix , b)
	
		if err != nil {
			t.Fatal(err)
		}
	}



// Get and Check
	for _ , v := range(tests) {

		val , err := rc.Get(*ss , v.Suffix)

		if err != nil {
			t.Fatal(err)
		}

		out := TestStoreValue{}
		err = out.Decode(val.([]byte))

		if err != nil {
			t.Fatal(err)
		}

		if out != v.TestVal  {
			t.Fatal("一致していません。")
		}
	}
}






func TestDelete(t *testing.T) {

	type Input struct {
		Suffix string
		TestVal TestStoreValue
	}


	rc := NewClient(getConninfo())

	ss , err := rc.NewSessionId()

	if err != nil {
		t.Fatal(err)
	}

	tests := []Input{
		Input{"testsuffix1" , TestStoreValue{TestId : "123" , TestValue : "XXXXAAAA"}},
		Input{"testsuffix2" , TestStoreValue{TestId : "234" , TestValue : "XXXXBBBB"}},
		Input{"testsuffix3" , TestStoreValue{TestId : "345" , TestValue : "XXXXXCCCC"}},
	}

	for _ , v := range(tests) {

		b , err := v.TestVal.Encode()

		if err != nil {
			t.Fatal(err)
		}

		err = rc.Set(*ss , v.Suffix , b)

		if err != nil {
			t.Fatal(err)
		}

	}

	//　excludeidxは、他のSuffix削除時に消えないことを確認するためのSuffix
	excludeidx := 1 
	idxs := []int{0,2}

	for _ , idx := range(idxs) {

		v := tests[idx]

		err =  rc.Delete(*ss , v.Suffix)

		if err != nil {
			t.Fatal(err)
		}

		//　消えていることを確認
		_ , err := rc.Get(*ss , v.Suffix)

		if err == nil {
			t.Fatal("存在しないはずのSuffixがエラーになっていません。")
		}

		if !errors.Is(err , ErrorSuffixNotFound) {
			t.Fatal(err)
		}

		// 存在しないSuffixを消そうとしたらエラー
		err =  rc.Delete(*ss , v.Suffix)

		if err == nil {
			t.Fatal("存在しないはずのSuffixがエラーになっていません。")
		}

		if !errors.Is(err , ErrorSuffixNotFound) {
			t.Fatal(err)
		}

		// 消えるはずないものが消えていないことを確認
		excludeval := tests[excludeidx]

		val , err := rc.Get(*ss , excludeval.Suffix)
		
		if err != nil {
			t.Fatal(err)
		}

		out := TestStoreValue{}
		err = out.Decode(val.([]byte))

		if err != nil {
			t.Fatal(err)
		}

		if out != excludeval.TestVal  {
			t.Fatal("一致していません。")
		}
	}

	excludeval := tests[excludeidx]

	err =  rc.Delete(*ss , excludeval.Suffix)

	if err != nil {
		t.Fatal(err)
	}

	err =  rc.Delete(*ss , excludeval.Suffix)

	if err == nil {
		t.Fatal("存在しないはずのSuffixがエラーになっていません。")
	}

	if !errors.Is(err , ErrorSuffixNotFound) {
		t.Fatal(err)
	}



}



func TestDestroy(t *testing.T) {

	rc := NewClient(getConninfo())

	ss , err := rc.NewSessionId()

	if err != nil {
		t.Fatal(err)
	}

	// 存在しないキーの削除はエラーにならない
	err = rc.Destroy(fmt.Sprintf("%sa",*ss))

	if err != nil {
		t.Fatal(err)
	}

	// 存在するキーの削除が成功する
	err = rc.Destroy(*ss)

	if err != nil {
		t.Fatal(err)
	}


}