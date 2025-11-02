package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
)

// KafkaProducer Kafka生产者
type KafkaProducer struct {
	writer *kafka.Writer
	mutex  sync.Mutex
}

// NewKafkaProducer 创建Kafka生产者
func NewKafkaProducer(brokers []string, topic string) *KafkaProducer {
	writer := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireAll,
		Async:        false,
		MaxAttempts:  3,
		WriteTimeout: 10 * time.Second,
	}

	return &KafkaProducer{
		writer: writer,
	}
}

// SendMessage 发送消息
func (p *KafkaProducer) SendMessage(message interface{}) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	var value []byte
	var err error

	// 尝试序列化消息
	if msgStr, ok := message.(string); ok {
		value = []byte(msgStr)
	} else {
		value, err = json.Marshal(message)
		if err != nil {
			return fmt.Errorf("failed to marshal message: %v", err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = p.writer.WriteMessages(ctx, kafka.Message{
		Value: value,
		Time:  time.Now(),
	})

	if err != nil {
		return fmt.Errorf("failed to send message: %v", err)
	}

	return nil
}

// SendMessageWithKey 发送带键的消息
func (p *KafkaProducer) SendMessageWithKey(key string, message interface{}) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	var value []byte
	var err error

	// 尝试序列化消息
	if msgStr, ok := message.(string); ok {
		value = []byte(msgStr)
	} else {
		value, err = json.Marshal(message)
		if err != nil {
			return fmt.Errorf("failed to marshal message: %v", err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(key),
		Value: value,
		Time:  time.Now(),
	})

	if err != nil {
		return fmt.Errorf("failed to send message with key: %v", err)
	}

	return nil
}

// Close 关闭生产者
func (p *KafkaProducer) Close() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	return p.writer.Close()
}

// KafkaConsumer Kafka消费者
type KafkaConsumer struct {
	reader    *kafka.Reader
	closeOnce sync.Once
}

// MessageHandler 消息处理函数类型
type MessageHandler func(message []byte) error

// NewKafkaConsumer 创建Kafka消费者
func NewKafkaConsumer(brokers []string, topic string, groupID string, offset int64) *KafkaConsumer {
	offsetConfig := kafka.FirstOffset
	if offset > 0 {
		offsetConfig = offset
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     brokers,
		Topic:       topic,
		GroupID:     groupID,
		MinBytes:    10e3, // 10KB
		MaxBytes:    10e6, // 10MB
		MaxWait:     10 * time.Second,
		StartOffset: offsetConfig,
	})

	return &KafkaConsumer{
		reader: reader,
	}
}

// Consume 消费消息
func (c *KafkaConsumer) Consume(ctx context.Context, handler MessageHandler) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// 读取消息
			msg, err := c.reader.ReadMessage(ctx)
			if err != nil {
				return fmt.Errorf("failed to read message: %v", err)
			}

			// 处理消息
			if err := handler(msg.Value); err != nil {
				// 记录错误但继续处理下一条消息
				fmt.Printf("failed to handle message: %v\n", err)
				continue
			}

			// 提交偏移量
			if err := c.reader.CommitMessages(ctx, msg); err != nil {
				fmt.Printf("failed to commit message: %v\n", err)
			}
		}
	}
}

// ConsumeBatch 批量消费消息
func (c *KafkaConsumer) ConsumeBatch(ctx context.Context, batchSize int, handler func([]kafka.Message) error) error {
	messages := make([]kafka.Message, 0, batchSize)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// 处理剩余消息
			if len(messages) > 0 {
				if err := handler(messages); err != nil {
					return fmt.Errorf("failed to handle batch: %v", err)
				}
				// 提交偏移量
				for _, msg := range messages {
					if err := c.reader.CommitMessages(ctx, msg); err != nil {
						fmt.Printf("failed to commit message: %v\n", err)
					}
				}
			}
			return ctx.Err()
		case <-ticker.C:
			// 定时处理批次
			if len(messages) > 0 {
				if err := handler(messages); err != nil {
					fmt.Printf("failed to handle batch: %v\n", err)
				} else {
					// 提交偏移量
					for _, msg := range messages {
						if err := c.reader.CommitMessages(ctx, msg); err != nil {
							fmt.Printf("failed to commit message: %v\n", err)
						}
					}
					// 清空批次
					messages = messages[:0]
				}
			}
		default:
			// 尝试读取消息但不阻塞
			ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
			msg, err := c.reader.ReadMessage(ctx)
			cancel()

			if err != nil {
				// 超时不算错误，继续等待
				if ctx.Err() == context.DeadlineExceeded {
					continue
				}
				return fmt.Errorf("failed to read message: %v", err)
			}

			messages = append(messages, msg)

			// 当批次满时处理
			if len(messages) >= batchSize {
				if err := handler(messages); err != nil {
					fmt.Printf("failed to handle batch: %v\n", err)
				} else {
					// 提交偏移量
					for _, msg := range messages {
						if err := c.reader.CommitMessages(ctx, msg); err != nil {
							fmt.Printf("failed to commit message: %v\n", err)
						}
					}
					// 清空批次
					messages = messages[:0]
				}
			}
		}
	}
}

// Close 关闭消费者
func (c *KafkaConsumer) Close() error {
	var err error
	c.closeOnce.Do(func() {
		err = c.reader.Close()
	})
	return err
}

// GetLag 获取消费者延迟
func (c *KafkaConsumer) GetLag(ctx context.Context) (map[int]int64, error) {
	partitions, err := c.reader.Partitions()
	if err != nil {
		return nil, fmt.Errorf("failed to get partitions: %v", err)
	}

	lagMap := make(map[int]int64)

	for _, p := range partitions {
		// 获取最新偏移量
		latestOffset, err := c.reader.ReadLag(ctx, p)
		if err != nil {
			return nil, fmt.Errorf("failed to get lag for partition %d: %v", p, err)
		}
		lagMap[int(p)] = latestOffset
	}

	return lagMap, nil
}
