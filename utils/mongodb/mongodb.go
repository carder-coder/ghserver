package mongodb

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/dobyte/due/v2/core/pool"
	"github.com/dobyte/due/v2/etc"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var factory = pool.NewFactory(func(name string) (*Client, error) {
	return NewInstance("etc.mongo")
})

type (
	Client = mongo.Client
)

type Config struct {
	URI      string `json:"dsn"`      // 连接串
	Database string `json:"database"` // 数据库名
}

// Instance 获取实例
func Instance(name ...string) *Client {
	var (
		err error
		ins *Client
	)

	if len(name) == 0 {
		ins, err = factory.Get("default")
	} else {
		ins, err = factory.Get(name[0])
	}

	if err != nil {
		log.Fatalf("create mongo instance failed: %v", err)
	}

	return ins
}

// NewInstance 新建实例
func NewInstance[T string | Config | *Config](config T) (*Client, error) {
	var (
		conf *Config
		v    any = config
		ctx      = context.Background()
	)

	switch c := v.(type) {
	case string:
		conf = &Config{URI: etc.Get(fmt.Sprintf("%s.uri", c)).String(),
			Database: etc.Get(fmt.Sprintf("%s.database", c)).String()}
	case Config:
		conf = &c
	case *Config:
		conf = c
	}

	opts := options.Client().ApplyURI(conf.URI)

	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		return nil, err
	}
	return client, nil
}

// MongoDBClient MongoDB客户端
type MongoDBClient struct {
	client     *mongo.Client
	database   string
	collection string
}

// NewMongoDBClient 创建MongoDB客户端
func NewMongoDBClient(database string, collection string) (*MongoDBClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client := Instance(database)
	// 测试连接
	err := client.Ping(ctx, readpref.Primary())
	if err != nil {
		return nil, fmt.Errorf("failed to ping mongodb: %v", err)
	}
	return &MongoDBClient{
		client:     client,
		database:   database,
		collection: collection,
	}, nil
}

// Close 关闭MongoDB连接
func (m *MongoDBClient) Close() error {
	return m.client.Disconnect(context.Background())
}

// GetCollection 获取集合
func (m *MongoDBClient) GetCollection() *mongo.Collection {
	return m.client.Database(m.database).Collection(m.collection)
}

// SetCollection 设置当前集合
func (m *MongoDBClient) SetCollection(collection string) {
	m.collection = collection
}

// InsertOne 插入单条数据
func (m *MongoDBClient) InsertOne(document interface{}) (interface{}, error) {
	result, err := m.GetCollection().InsertOne(context.Background(), document)
	if err != nil {
		return nil, err
	}
	return result.InsertedID, nil
}

// InsertMany 插入多条数据
func (m *MongoDBClient) InsertMany(documents []interface{}) ([]interface{}, error) {
	result, err := m.GetCollection().InsertMany(context.Background(), documents)
	if err != nil {
		return nil, err
	}
	return result.InsertedIDs, nil
}

// FindOne 查询单条数据
func (m *MongoDBClient) FindOne(filter interface{}, result interface{}) error {
	return m.GetCollection().FindOne(context.Background(), filter).Decode(result)
}

// Find 查询多条数据
func (m *MongoDBClient) Find(filter interface{}, results interface{}, limit int64, skip int64) error {
	options := options.Find()
	if limit > 0 {
		options.SetLimit(limit)
	}
	if skip > 0 {
		options.SetSkip(skip)
	}

	cursor, err := m.GetCollection().Find(context.Background(), filter, options)
	if err != nil {
		return err
	}
	defer cursor.Close(context.Background())

	return cursor.All(context.Background(), results)
}

// UpdateOne 更新单条数据
func (m *MongoDBClient) UpdateOne(filter interface{}, update interface{}) error {
	_, err := m.GetCollection().UpdateOne(context.Background(), filter, update)
	return err
}

// UpdateMany 更新多条数据
func (m *MongoDBClient) UpdateMany(filter interface{}, update interface{}) (int64, error) {
	result, err := m.GetCollection().UpdateMany(context.Background(), filter, update)
	if err != nil {
		return 0, err
	}
	return result.ModifiedCount, nil
}

// DeleteOne 删除单条数据
func (m *MongoDBClient) DeleteOne(filter interface{}) error {
	_, err := m.GetCollection().DeleteOne(context.Background(), filter)
	return err
}

// DeleteMany 删除多条数据
func (m *MongoDBClient) DeleteMany(filter interface{}) (int64, error) {
	result, err := m.GetCollection().DeleteMany(context.Background(), filter)
	if err != nil {
		return 0, err
	}
	return result.DeletedCount, nil
}

// CountDocuments 统计文档数量
func (m *MongoDBClient) CountDocuments(filter interface{}) (int64, error) {
	return m.GetCollection().CountDocuments(context.Background(), filter)
}

// Aggregate 聚合查询
func (m *MongoDBClient) Aggregate(pipeline interface{}, results interface{}) error {
	cursor, err := m.GetCollection().Aggregate(context.Background(), pipeline)
	if err != nil {
		return err
	}
	defer cursor.Close(context.Background())

	return cursor.All(context.Background(), results)
}

// FindOneAndUpdate 查询并更新
func (m *MongoDBClient) FindOneAndUpdate(filter interface{}, update interface{}, result interface{}) error {
	return m.GetCollection().FindOneAndUpdate(context.Background(), filter, update).Decode(result)
}

// FindOneAndReplace 查询并替换
func (m *MongoDBClient) FindOneAndReplace(filter interface{}, replacement interface{}, result interface{}) error {
	return m.GetCollection().FindOneAndReplace(context.Background(), filter, replacement).Decode(result)
}

// FindOneAndDelete 查询并删除
func (m *MongoDBClient) FindOneAndDelete(filter interface{}, result interface{}) error {
	return m.GetCollection().FindOneAndDelete(context.Background(), filter).Decode(result)
}

// EnsureIndex 确保索引存在
func (m *MongoDBClient) EnsureIndex(indexName string, keys interface{}, unique bool) error {
	idxOptions := options.Index()
	idxOptions.SetName(indexName)
	idxOptions.SetUnique(unique)

	indexModel := mongo.IndexModel{
		Keys:    keys,
		Options: idxOptions,
	}

	_, err := m.GetCollection().Indexes().CreateOne(context.Background(), indexModel)
	return err
}

// DropIndex 删除索引
func (m *MongoDBClient) DropIndex(indexName string) error {
	return m.GetCollection().Indexes().DropOne(context.Background(), indexName)
}

// ListIndexes 列出所有索引
func (m *MongoDBClient) ListIndexes() ([]string, error) {
	cursor, err := m.GetCollection().Indexes().List(context.Background())
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())

	var indexes []string
	for cursor.Next(context.Background()) {
		var idx bson.M
		if err := cursor.Decode(&idx); err != nil {
			return nil, err
		}
		if name, ok := idx["name"].(string); ok {
			indexes = append(indexes, name)
		}
	}

	return indexes, nil
}

// GetDatabaseStats 获取数据库统计信息
func (m *MongoDBClient) GetDatabaseStats() (bson.M, error) {
	var stats bson.M
	err := m.client.Database(m.database).RunCommand(context.Background(), bson.D{{Key: "dbStats", Value: 1}}).Decode(&stats)
	return stats, err
}

// GetCollectionStats 获取集合统计信息
func (m *MongoDBClient) GetCollectionStats() (bson.M, error) {
	var stats bson.M
	err := m.client.Database(m.database).RunCommand(context.Background(), bson.D{{Key: "collStats", Value: m.collection}}).Decode(&stats)
	return stats, err
}
