package globalTypes

import "strconv"

func NewItemFromString(data string) ItemSoftIdentity {
	number, err := strconv.ParseUint(data, 10, 64)
	if err != nil {
		return ItemSoftIdentity{
			ItemName: data,
		}
	}
	return ItemSoftIdentity{
		ItemId: uint(number),
	}
}

func NewRealmFromString(data string) ConnectedRealmSoftIentity {
	number, err := strconv.ParseUint(data, 10, 64)
	if err != nil {
		return ConnectedRealmSoftIentity{
			Name: data,
		}
	}
	return ConnectedRealmSoftIentity{
		Id: uint(number),
	}
}
