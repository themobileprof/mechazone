package ledger

import "testing"

func TestInShopScope(t *testing.T) {
	if !InShopScope("shop-a", "shop-a", "tech-1", "tech-9") {
		t.Fatal("same shop must see its own job")
	}
	if InShopScope("shop-a", "shop-b", "tech-1", "tech-1") {
		t.Fatal("other shop's job must stay hidden")
	}
	if !InShopScope("", "", "tech-1", "tech-1") {
		t.Fatal("freelancer sees own job")
	}
	if InShopScope("", "shop-a", "tech-1", "tech-1") {
		t.Fatal("freelancer must not see shop jobs")
	}
	if InShopScope("", "", "tech-1", "tech-2") {
		t.Fatal("freelancer must not see another freelancer")
	}
}
