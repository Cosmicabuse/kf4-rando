package shops

import (
	"encoding/binary"
	"fmt"
	"math/rand/v2"

	"kf4randomizer/internal/kfconst"
)

// NewRand salts the seed so each pass gets its own stream.
func NewRand(seed uint64) *rand.Rand {
	return rand.New(rand.NewPCG(seed, seed^0x94d049bb133111eb))
}

// since some checks are locked behind projectiles,
// base projectiles should be made available in every shop if randomizer is on.
// The GetItem routine is too hard to modify to, for ex., give # arrows when the first Bow is picked up.
// Since gold isn't randomized, I think it works out. Prices are loosely based on vanilla.
var ammo = []struct {
	id  int
	buy uint16
}{
	{145, 20},  // Arrow
	{146, 20},  // Bolt
	{147, 200}, // Widda's Arrow
}

// shops can sell unlimited with 255 quantity specified
const unlimited = 255

// Sale is one item a shop sells.
type Sale struct {
	ID       uint16
	Quantity byte
	Buy      uint16
}

// Result lists every gold shop's stock, index = shop.
type Result struct {
	Shops [][]Sale
}

// Shuffle pools each item once, however many shops sold it, and deals them
// back with shops shrunk to match. A moved item costs what its new shop asks.
func Shuffle(stock []byte, rng *rand.Rand) error {
	if len(stock) != kfconst.ShopCount*kfconst.ShopStockSize {
		return fmt.Errorf("stock is %d bytes, want %d", len(stock), kfconst.ShopCount*kfconst.ShopStockSize)
	}
	// ammo lookup for later
	isAmmo := map[int]bool{}
	for _, a := range ammo {
		isAmmo[a.id] = true
	}

	var pool []int
	seen := map[int]bool{}
	counts := make([]int, kfconst.ShopGoldCount)
	for shop := range kfconst.ShopGoldCount {
		for id := 1; id < kfconst.ShopStockSize; id++ {
			if stockAt(stock, shop, id) == 0 || isAmmo[id] {
				continue
			}
			counts[shop]++
			stock[shop*kfconst.ShopStockSize+id] = 0
			if !seen[id] {
				seen[id] = true
				pool = append(pool, id)
			}
		}
	}

	rng.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })
	next := 0
	for shop, size := range shareOut(counts, len(pool)) {
		for _, id := range pool[next : next+size] {
			qty := byte(1)
			if common(id) {
				qty = unlimited
			}
			stock[shop*kfconst.ShopStockSize+id] = qty
		}
		next += size
	}
	return nil
}

// common reports whether a shop sells an item without limit: Herb through
// Dragon Flower, and the Power Seed and Widda Needle, which wear off.
func common(id int) bool {
	return (id >= 6 && id <= 13) || id == 21 || id == 22
}

// shareOut splits n items over shops in proportion to their vanilla counts,
// handing the leftovers to the largest remainders.
func shareOut(counts []int, n int) []int {
	total := 0
	for _, c := range counts {
		total += c
	}
	sizes := make([]int, len(counts))
	if total == 0 {
		return sizes
	}
	left := n
	rest := make([]int, len(counts))
	for i, c := range counts {
		sizes[i] = c * n / total
		rest[i] = c * n % total
		left -= sizes[i]
	}
	for ; left > 0; left-- {
		best := 0
		for i := range rest {
			if rest[i] > rest[best] {
				best = i
			}
		}
		sizes[best]++
		rest[best] = -1
	}
	return sizes
}

// Clear empties the gold shops for a new stock.
func Clear(stock []byte) {
	for shop := range kfconst.ShopGoldCount {
		clear(stock[shop*kfconst.ShopStockSize : (shop+1)*kfconst.ShopStockSize])
	}
}

// Add puts one more of id in a gold shop, or makes it unlimited if common.
func Add(stock []byte, shop int, id uint16) {
	i := shop*kfconst.ShopStockSize + int(id)
	switch {
	case common(int(id)):
		stock[i] = unlimited
	case stock[i] < unlimited-1:
		stock[i]++
	}
}

// AddAmmo stocks every gold shop with all three ammo types, unlimited.
func AddAmmo(stock, prices []byte) error {
	if err := check(stock, prices); err != nil {
		return err
	}
	for shop := range kfconst.ShopGoldCount {
		for _, a := range ammo {
			stock[shop*kfconst.ShopStockSize+a.id] = unlimited
			_, sell := priceAt(prices, shop, a.id)
			setPrice(prices, shop, a.id, a.buy, sell)
		}
	}
	return nil
}

// List reads back what every gold shop sells.
func List(stock, prices []byte) Result {
	res := Result{Shops: make([][]Sale, kfconst.ShopGoldCount)}
	for shop := range kfconst.ShopGoldCount {
		for id := 1; id < kfconst.ShopStockSize; id++ {
			if q := stockAt(stock, shop, id); q != 0 {
				buy, _ := priceAt(prices, shop, id)
				res.Shops[shop] = append(res.Shops[shop], Sale{uint16(id), q, buy})
			}
		}
	}
	return res
}

func check(stock, prices []byte) error {
	if len(stock) != kfconst.ShopCount*kfconst.ShopStockSize {
		return fmt.Errorf("stock is %d bytes, want %d", len(stock), kfconst.ShopCount*kfconst.ShopStockSize)
	}
	if len(prices) != kfconst.ShopCount*kfconst.ShopPriceSize {
		return fmt.Errorf("prices is %d bytes, want %d", len(prices), kfconst.ShopCount*kfconst.ShopPriceSize)
	}
	return nil
}

// Quantity is how many of id a shop holds; unlimited counts as one.
func Quantity(stock []byte, shop int, id uint16) int {
	if q := stockAt(stock, shop, int(id)); q != unlimited {
		return int(q)
	}
	return 1
}

func stockAt(stock []byte, shop, id int) byte {
	return stock[shop*kfconst.ShopStockSize+id]
}

func priceAt(prices []byte, shop, id int) (buy, sell uint16) {
	p := prices[shop*kfconst.ShopPriceSize+id*4:]
	return binary.LittleEndian.Uint16(p), binary.LittleEndian.Uint16(p[2:])
}

func setPrice(prices []byte, shop, id int, buy, sell uint16) {
	p := prices[shop*kfconst.ShopPriceSize+id*4:]
	binary.LittleEndian.PutUint16(p, buy)
	binary.LittleEndian.PutUint16(p[2:], sell)
}
