package ledger

import "testing"

func TestActionTokensMorphDLCVoltage(t *testing.T) {
	a := ActionTokens("test", "Prove battery on pin 16 of the DLC")
	b := ActionTokens("Test", "Check DLC pin 16 battery voltage")
	if !ShouldMerge("test", "test", a, b) {
		t.Fatalf("should merge %v vs %v j=%.2f", a, b, Jaccard(a, b))
	}
	if ActionFingerprint("test", a) == "" {
		t.Fatal("fingerprint")
	}
}

func TestActionTokensOhmVsVoltStayApart(t *testing.T) {
	ohm := ActionTokens("test", "Measure injector resistance")
	volt := ActionTokens("test", "Measure injector supply voltage")
	if ShouldMerge("test", "test", ohm, volt) {
		t.Fatalf("ohm and volt must not merge: %v vs %v", ohm, volt)
	}
}

func TestActionTokensClearCodesMerge(t *testing.T) {
	a := ActionTokens("test", "Clear stored codes")
	b := ActionTokens("test", "Clear DTCs then road test")
	if !ShouldMerge("test", "test", a, b) {
		t.Fatalf("clear should merge %v vs %v j=%.2f", a, b, Jaccard(a, b))
	}
}

func TestActionTokensBackprobeMerge(t *testing.T) {
	a := ActionTokens("access", "Backprobe the connector")
	b := ActionTokens("access", "Land from the rear of the connector")
	if !ShouldMerge("access", "access", a, b) {
		t.Fatalf("backprobe should merge %v vs %v j=%.2f", a, b, Jaccard(a, b))
	}
}

func TestActionTokensKindMustMatch(t *testing.T) {
	a := ActionTokens("test", "Measure DC voltage")
	b := ActionTokens("inspect", "Measure DC voltage")
	if ShouldMerge("test", "inspect", a, b) {
		t.Fatal("different kinds must not merge")
	}
}

func TestJaccardIdentical(t *testing.T) {
	tok := ActionTokens("test", "Continuity beep on the two points")
	if Jaccard(tok, tok) != 1 {
		t.Fatal(Jaccard(tok, tok))
	}
}
