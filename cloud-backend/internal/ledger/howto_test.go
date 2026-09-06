package ledger

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestObservePlaybookStepsMorphsSimilarTitles(t *testing.T) {
	s := testStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	if err := s.SeedHowTos(ctx); err != nil {
		t.Fatal(err)
	}
	tag := fmt.Sprintf("DLC batt %d", time.Now().UnixNano()%1e9)
	first, err := s.ObserveAction(ctx, "test", "Prove battery on pin 16 of the DLC "+tag)
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.ObserveAction(ctx, "test", "Check DLC pin 16 battery voltage "+tag)
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != second.ID {
		t.Fatalf("expected merge %s vs %s (%v / %v)", first.ID, second.ID, first.Tokens, second.Tokens)
	}
	if second.SeenCount < 2 {
		t.Fatalf("seen %d", second.SeenCount)
	}
	guides, err := s.ListHowToGuides(ctx, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(guides) < 5 {
		t.Fatalf("seeded %d", len(guides))
	}
	g, err := s.CreateHowToGuide(ctx, HowToIn{
		Title: "Temp card " + tag, Blurb: "b", Warning: "w",
		BodyHTML:   `<p>Hands on the meter.</p><img src="/howto/meter-volts-dial.jpg" alt="dial">`,
		MatchWords: []string{"temp-card-" + tag},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.DeleteHowToGuide(context.Background(), g.ID) })
}

func TestCreateHowToRejectsEmptyTitle(t *testing.T) {
	s := testStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	if _, err := s.CreateHowToGuide(ctx, HowToIn{BodyHTML: "<p>x</p>"}); err == nil {
		t.Fatal("expected title error")
	}
}
