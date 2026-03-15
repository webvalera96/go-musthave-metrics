package pool

import (
	"reflect"
	"sync"
)

// Resetter ограничивает типы, которые должны иметь метод Reset()
type Resetter interface {
	Reset()
}

// Pool представляет собой пул объектов с generic-параметром.
// Тип T должен реализовывать интерфейс Resetter (иметь метод Reset()).
// Перед возвратом объекта в пул через метод Put() автоматически вызывается метод Reset().
type Pool[T Resetter] struct {
	pool sync.Pool
}

// New создает и возвращает новый пул объектов типа T.
// Функция new создает новый экземпляр объекта, если пул пуст.
//
// Пример использования:
//
//	type MyStruct struct {
//	    value int
//	}
//
//	func (m *MyStruct) Reset() {
//	    m.value = 0
//	}
//
//	p := pool.New[*MyStruct](func() *MyStruct {
//	    return &MyStruct{}
//	})
func New[T Resetter](new func() T) *Pool[T] {
	return &Pool[T]{
		pool: sync.Pool{
			New: func() interface{} {
				return new()
			},
		},
	}
}

// Get возвращает объект из пула.
// Если пул пуст, создается новый объект с помощью функции new, переданной в конструктор.
// Возвращаемый объект уже сброшен (метод Reset() был вызван при предыдущем Put()).
func (p *Pool[T]) Get() T {
	return p.pool.Get().(T)
}

// Put помещает объект обратно в пул.
// Перед помещением объекта в пул автоматически вызывается метод Reset() для сброса состояния объекта.
// Важно: после вызова Put() не следует использовать объект, так как он может быть переиспользован другим кодом.
// Если x является указателем и равен nil, метод ничего не делает.
func (p *Pool[T]) Put(x T) {
	// Проверяем, что объект не nil (для указателей и интерфейсов)
	v := reflect.ValueOf(x)
	if v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return
		}
	}
	// Вызываем Reset() перед возвратом в пул
	x.Reset()
	p.pool.Put(x)
}
