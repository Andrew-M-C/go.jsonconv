package jsonconv
import ()

func (to *JsonValue) copyFrom(from *JsonValue) {
	to.valueType = from.valueType
	to.stringValue = from.stringValue
	to.intValue = from.intValue
	to.floatValue = from.floatValue
	to.boolValue = from.boolValue
	to.objChildren = from.objChildren
	to.arrChildren = from.arrChildren
}

func (to *JsonValue) MergeFrom(from *JsonValue, optList ...Option) error {
	if nil == from {
		return nil
	}

	var opt *Option
	if len(optList) == 0 {
		opt = &dftOption
	} else {
		opt = &optList[0]
	}
	should_override_array := opt.OverrideArray

	switch to.valueType {
	case String, Number, Boolean, Null:
		// if value is of a basic type, simply override it
		to.copyFrom(from)
	case Object:
		if from.valueType != Object {
			// just override the whole object
			to.copyFrom(from)
		} else if opt.OverrideObject {
			to.copyFrom(from)
		} else {
			// go throuth each child
			from.ObjectForeach(func(key string, value *JsonValue) error {
				to_child, _ := to.Get(key)
				if nil == to_child {
					to.Set(value, key)
				} else {
					to_child.MergeFrom(value, *opt)
				}
				return nil
			})
		}
	case Array:
		if from.valueType != Array {
			// just override the whole object
			to.copyFrom(from)
		} else if should_override_array {
			to.copyFrom(from)
		} else {
			// append
			to.arrChildren = append(to.arrChildren, from.arrChildren...)
		}
	default:
		return JsonTypeError
	}

	return nil
}
