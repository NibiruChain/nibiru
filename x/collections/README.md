# Collections

A simplified way to deal with Cosmos-SDK module storage.

Collections has multiple APIs to deal with different storage kinds. It offers complex iteration APIs which support even multi-part keys.

It's pure golang, no reflection, no protobuf options, only generics.

Each collections API accepts a KeyEncoder and (or) a ValueEncoder.

## Examples

Examples are [here](./examples). Follow them in order.

## Namespaces

Each collection type will expect you to define a namespace, a namespace is a number which ranges from 0 to 255.
Each collection type (Indexers too), must have a unique namespace in the module.

## KeyEncoders

KeyEncoder teaches to collection how to encode and decode the key used to map the object into the storage.

Collections comes in with a preset of key encoders which guarantee lexographical ordering of keys, more can be added depending on your needs as long as you implement the KeyEncoder interface.


# ValueEncoders

ValueEncoder teaches the collection type how to convert the object we're storing into bytes, or turning the bytes
into the object stored itself.



## Map

Map is the first storage type which maps keys to object.
Objects can be of different kinds.

Example:

````go
type MyKeeper struct {
	Balances collections.Map[sdk.AccAddress, sdk.Coins]
}

func (k MyKeeper) SendCoin(ctx sdk.Context, from, to sdk.AccAddress, coin sdk.Coins) error {
	balance, err := k.Balances.Get(ctx, from)
	if err != nil {
		return err
    }
	
	newBalance, ok := balance.SafeSub(coin)
	if !ok { return fmt.Errorf("not enough balance") }
	k.Balances.Insert(ctx, from, newBalance)
	...
}
````


### KeySet

KeySet, as the words says, is a set of keys. It maps no objects but it retains a set of keys.

Example:

```go
type MyKeeper struct {
	AllowList collections.KeySet[sdk.AccAddress]
}

func (m MyKeeper) IsAllowed(ctx sdk.Context, addr sdk.AccAddress) bool {
	return m.AllowList.Has(ctx, addr)
}

func (m MyKeeper) Allow(ctx sdk.Context, addr sdk.AccAddress) {
	m.AllowList.Insert(ctx, addr)
}

func (m MyKeeper) Disallow(ctx sdk.Context, addr sdk.AccAddress) {
	m.AllowList.Delete(ctx, addr)
}
```

## Item

Item is a collection type which contains only one object, it's usually used for configs, sequences etc.

````go
type MyKeeper struct {
	Config collections.Item[Config]
}

func (m MyKeeper) UpdateConfig(ctx sdk.Context, conf config) {
	m.Config.Set(ctx, conf)
}

func (m MyKeeper) GetConfig(ctx sdk.Context) (Config, err) {
	return m.Config.Get(ctx)
}
````

## Sequence

Sequence is a helper type which implements an ever increasing number.


## IndexedMap 

IndexedMap allows to create complex indexing functionalities around stored objects.
It builds on top of a normal collections.Map and uses collections.KeySet to create reference keys
between the primary key and the indexing key.
