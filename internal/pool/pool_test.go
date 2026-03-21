package pool

import (
	"testing"
)

// TestStruct - тестовая структура с методом Reset()
type TestStruct struct {
	Value int
	Data  string
}

func (ts *TestStruct) Reset() {
	ts.Value = 0
	ts.Data = ""
}

func TestPool_GetPut(t *testing.T) {
	// Создаем пул
	p := New[*TestStruct](func() *TestStruct {
		return &TestStruct{}
	})

	// Получаем объект из пула
	obj1 := p.Get()
	if obj1 == nil {
		t.Fatal("Get() returned nil")
	}

	// Изменяем состояние объекта
	obj1.Value = 42
	obj1.Data = "test"

	// Возвращаем объект в пул
	p.Put(obj1)

	// Получаем объект снова
	obj2 := p.Get()

	// Проверяем, что состояние сброшено
	if obj2.Value != 0 {
		t.Errorf("Expected Value to be 0 after Reset(), got %d", obj2.Value)
	}
	if obj2.Data != "" {
		t.Errorf("Expected Data to be empty after Reset(), got %s", obj2.Data)
	}

	// Проверяем, что это тот же объект (переиспользование)
	if obj1 != obj2 {
		t.Log("Note: Different objects returned, which is fine for sync.Pool behavior")
	}
}

func TestPool_PutNil(t *testing.T) {
	p := New[*TestStruct](func() *TestStruct {
		return &TestStruct{}
	})

	// Проверяем, что Put(nil) не вызывает панику
	p.Put(nil)

	// Получаем объект - должен быть новый, не nil
	obj := p.Get()
	if obj == nil {
		t.Fatal("Get() returned nil after Put(nil)")
	}
}

func TestPool_MultipleGets(t *testing.T) {
	p := New[*TestStruct](func() *TestStruct {
		return &TestStruct{Value: 100}
	})

	// Получаем несколько объектов
	obj1 := p.Get()
	obj2 := p.Get()

	if obj1 == obj2 {
		t.Error("Expected different objects from Get()")
	}

	// Возвращаем оба в пул
	p.Put(obj1)
	p.Put(obj2)

	// Получаем снова - должны получить сброшенные объекты
	obj3 := p.Get()
	if obj3.Value != 0 {
		t.Errorf("Expected Value to be 0 after Reset(), got %d", obj3.Value)
	}
}
