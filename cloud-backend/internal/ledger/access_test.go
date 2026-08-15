package ledger

import "testing"

func TestNormalizeAccessRequest(t *testing.T) {
	ok, err := NormalizeAccessRequest(CreateAccessRequestInput{
		ApplicantName: "  Ada  Okonkwo ",
		ContactEmail:  "Ada@Yaba.Test",
		ContactPhone:  "08031234567",
		City:          "Lagos",
		Kind:          "shop",
		ShopName:      "Yaba Motors",
	})
	if err != nil {
		t.Fatal(err)
	}
	if ok.ContactEmail != "ada@yaba.test" || ok.ApplicantName != "Ada Okonkwo" || ok.Country != "Nigeria" {
		t.Fatalf("%+v", ok)
	}

	_, err = NormalizeAccessRequest(CreateAccessRequestInput{
		ApplicantName: "Ada", ContactEmail: "not-an-email", ContactPhone: "08031234567", City: "Lagos", Kind: "shop", ShopName: "Bay",
	})
	if err == nil {
		t.Fatal("expected bad email")
	}

	_, err = NormalizeAccessRequest(CreateAccessRequestInput{
		ApplicantName: "Ada", ContactEmail: "ada@yaba.test", ContactPhone: "08031234567", City: "Lagos", Kind: "shop",
	})
	if err == nil {
		t.Fatal("expected shop name")
	}

	free, err := NormalizeAccessRequest(CreateAccessRequestInput{
		ApplicantName: "Ada", ContactEmail: "ada@yaba.test", ContactPhone: "08031234567", City: "Lagos", Kind: "freelancer",
	})
	if err != nil {
		t.Fatal(err)
	}
	if free.Kind != "freelancer" {
		t.Fatalf("kind %s", free.Kind)
	}

	_, err = NormalizeAccessRequest(CreateAccessRequestInput{
		ApplicantName: "Ada", ContactEmail: "ada@yaba.test", City: "Lagos", Kind: "freelancer",
	})
	if err == nil {
		t.Fatal("expected WhatsApp")
	}
}
