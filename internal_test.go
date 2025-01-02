package jsonrpc2

func NewMessageListForTest[T any]() messageList[T] {
	return messageList[T]{}
}
