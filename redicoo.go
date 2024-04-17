package redicoo

import (
	"fmt"
	"time"
	"errors"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

var ErrorKeyNotFound = errors.New("Key is not found")
var ErrorSuffiedKeyNotFound = errors.New("suffixed Key is not found")
var ErrorFailed = errors.New("update failed")

type ConnInfo struct {
	Addr string
	DB int
	Lifetime *int64
}


type RedisClient struct {
	options redis.Options
}

type StoreValue interface {
	Encode() ([]byte , error)
	Decode(input []byte) error
}

const matRetries = 10

var lifetime int64 = 600

// func initLifetime(lifesec int64) {
// 	lifetime = lifesec
// }



func NewClient(opt ConnInfo) RedisClient {

	ropt := redis.Options{
		Addr: opt.Addr,
		DB:   opt.DB,		
	}

	if opt.Lifetime != nil {
		lifetime = *opt.Lifetime
	}

	return RedisClient{ropt}
}



func (rc RedisClient) connect(ctx context.Context) (*redis.Client , error) {

	client := redis.NewClient(&rc.options)

	_,err := client.Ping(ctx).Result()

	if err != nil {
		return nil , err
	}

	return client , nil
}


func (rc RedisClient) NewSessionId() (*string , error) {
	
	v7 , err := uuid.NewV7()

	if err != nil {
		return nil , err
	}

	uuidstr := v7.String()

	r := sha256.Sum256([]byte(uuidstr))

	keystr:= hex.EncodeToString(r[:])

	ctx := context.Background()

	client , err := rc.connect(ctx)

	defer client.Close()

	if err != nil {
		return nil , err
	}

	err = client.Set(ctx , keystr , nil , time.Duration(lifetime) * time.Second).Err()

	if err != nil {
		return nil , err
	}

	return &keystr , nil
}



func (rc RedisClient) IsExists(key string) (bool , error){

	ctx := context.Background()

	client , err := rc.connect(ctx)

	if err != nil {
		return false , err
	}

	defer client.Close()

	cmds, err := client.Pipelined(ctx, func(pipe redis.Pipeliner) error {

		duration := time.Duration(lifetime) * time.Second
		
		pipe.Get(ctx , key)
		pipe.Expire(ctx, key, duration)

		return nil
	})


	if err == redis.Nil {
		return false , fmt.Errorf("%w",ErrorKeyNotFound)
	}

	if err != nil {
		return false , err
	}

	exp , err := cmds[1].(*redis.BoolCmd).Result()

	if !exp {
		return false , fmt.Errorf("%w",ErrorFailed)
	}

	return true , nil
}




func (rc RedisClient) Set(key string , suffix string , value []byte) error {

	ctx := context.Background()

	client , err := rc.connect(ctx)

	if err != nil {
		return err
	}

	defer client.Close()



	txf:= func (tx *redis.Tx) error {

		res ,err := client.Get(ctx , key).Result()

		if err == redis.Nil {
			return fmt.Errorf("%w",ErrorKeyNotFound)
		}
	
		if err != nil {
			return err
		}
	
		kv := make(map[string][]byte)
	
		if len(res) >0 {
			
			err = json.Unmarshal([]byte(res) , &kv)
	
			if err != nil {
				return err
			}
		}
	
		kv[suffix] = value
	
		jsonData , err := json.Marshal(kv)
	
		if err != nil {
			return err
		}
	
		duration := time.Duration(lifetime) * time.Second
	
		err = client.Set(ctx, key , jsonData, duration).Err()
	
		if err!=nil {
			return err
		} 
	
		return nil
	}


	for i := 0 ; i<matRetries ; i++ {

		err = client.Watch(ctx, txf, key)

		if err == nil {
			return nil
		}
	
		if err == redis.TxFailedErr {
			continue
		}

		if err != nil {
			return err
		}
	}

	return fmt.Errorf("%w",ErrorFailed)

}







func (rc RedisClient) Get(key string ,suffix string) (interface{},error) {

	ctx := context.Background()

	client , err := rc.connect(ctx)

	if err != nil {
		return nil , err
	}

	defer client.Close()

	cmds, err := client.Pipelined(ctx, func(pipe redis.Pipeliner) error {

		duration := time.Duration(lifetime) * time.Second
		
		pipe.Get(ctx , key)
		pipe.Expire(ctx, key, duration)

		return nil
	})

	if err == redis.Nil {
		return nil , fmt.Errorf("%w",ErrorKeyNotFound)
	}

	if err != nil {
		return nil , err
	}

	exp , err := cmds[1].(*redis.BoolCmd).Result()

	if !exp {
		fmt.Errorf("%w",ErrorFailed)
	}


	res , err := cmds[0].(*redis.StringCmd).Result()

	if err != nil {
		return nil , err
	}

	if len(res) == 0 {
		return nil , fmt.Errorf("%w",ErrorSuffiedKeyNotFound)
	}

	kv := make(map[string][]byte)

	err = json.Unmarshal([]byte(res) , &kv)

	if err != nil {
		return nil , err
	}

	val , ok := kv[suffix]

	if !ok {
		return nil , fmt.Errorf("%w",ErrorSuffiedKeyNotFound)
	}

	return val , nil
}


func (rc RedisClient) Delete(key string ,suffix string) error {

	ctx := context.Background()

	client , err := rc.connect(ctx)

	if err != nil {
		return err
	}

	defer client.Close()


	txf:= func (tx *redis.Tx) error {


		res , err := client.Get(ctx , key).Result()

		if err == redis.Nil {
			return fmt.Errorf("%w",ErrorKeyNotFound)
		}

		if err != nil {
			return err
		}

		if len(res) == 0 {
			return fmt.Errorf("%w",ErrorSuffiedKeyNotFound)
		}


		kv := make(map[string][]byte)

		err = json.Unmarshal([]byte(res) , &kv)

		if err != nil {
			return err
		}


		_ , ok := kv[suffix]

		if !ok {
			return fmt.Errorf("%w",ErrorSuffiedKeyNotFound)
		}

		delete(kv, suffix)

		duration := time.Duration(lifetime) * time.Second

		if len(kv) == 0 {

			err = client.Set(ctx, key , nil, duration).Err()
	
			if err!=nil {
				return err
			} 

		} else {

			jsonData , err := json.Marshal(kv)

			if err != nil {
				return err
			}
	
			err = client.Set(ctx, key , jsonData, duration).Err()
	
			if err!=nil {
				return err
			} 
		}

		return nil
	}



	for i := 0 ; i<matRetries ; i++ {

		err = client.Watch(ctx, txf, key)

		if err == nil {
			return nil
		}
	
		if err == redis.TxFailedErr {
			continue
		}

		if err != nil {
			return err
		}
	}

	return fmt.Errorf("%w",ErrorFailed)

}


func (rc RedisClient) Destroy(key string) error {

	ctx := context.Background()

	client , err := rc.connect(ctx)

	if err != nil {
		return err
	}

	defer client.Close()

	_ , err = client.Del(ctx, key).Result()

	if err == redis.Nil {
		return nil
	}

	if err != nil {
		return err
	}

	return nil
}