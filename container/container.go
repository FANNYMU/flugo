package container

import (
	"fmt"
	"reflect"
)

type Container struct {
	providers map[string]interface{}
	instances map[string]interface{}
}

func NewContainer() *Container {
	return &Container{
		providers: make(map[string]interface{}),
		instances: make(map[string]interface{}),
	}
}

func (c *Container) Register(provider interface{}) {
	t := reflect.TypeOf(provider)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	name := t.String()
	c.providers[name] = provider
}

func (c *Container) Resolve(target interface{}) error {
	targetValue := reflect.ValueOf(target)
	if targetValue.Kind() != reflect.Ptr {
		return fmt.Errorf("target must be a pointer")
	}

	targetType := targetValue.Elem().Type()
	if targetType.Kind() != reflect.Struct {
		return fmt.Errorf("target must be a pointer to struct")
	}

	for i := 0; i < targetType.NumField(); i++ {
		field := targetType.Field(i)
		if field.Tag.Get("inject") == "true" {
			instance, err := c.getInstance(field.Type)
			if err != nil {
				return err
			}
			targetValue.Elem().Field(i).Set(reflect.ValueOf(instance))
		}
	}

	return nil
}

func (c *Container) getInstance(t reflect.Type) (interface{}, error) {
	typeName := t.String()

	if instance, exists := c.instances[typeName]; exists {
		return instance, nil
	}

	provider, exists := c.providers[typeName]
	if !exists {
		return nil, fmt.Errorf("provider not found for type: %s", typeName)
	}

	providerValue := reflect.ValueOf(provider)
	if providerValue.Kind() == reflect.Func {
		results := providerValue.Call(nil)
		if len(results) > 0 {
			instance := results[0].Interface()
			c.instances[typeName] = instance
			return instance, nil
		}
	} else {
		c.instances[typeName] = provider
		return provider, nil
	}

	return nil, fmt.Errorf("failed to create instance for type: %s", typeName)
}

func (c *Container) GetInstance(t reflect.Type) (interface{}, error) {
	return c.getInstance(t)
}
