package domain

import "testing"

func TestUserRoleIsValid(t *testing.T) {
	cases := []struct {
		role  UserRole
		valid bool
	}{
		{RoleRider, true},
		{RoleDriver, true},
		{RoleAdmin, true},
		{UserRole("HACKER"), false},
		{UserRole(""), false},
		{UserRole("rider"), false},
	}
	for _, tc := range cases {
		if got := tc.role.IsValid(); got != tc.valid {
			t.Errorf("role %q IsValid=%v want %v", tc.role, got, tc.valid)
		}
	}
}

func TestUserValidate(t *testing.T) {
	valid := &User{Email: "a@b.com", Phone: "+15551212", FullName: "Ada", Role: RoleRider}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid user rejected: %v", err)
	}

	missing := &User{Email: "", Phone: "1", FullName: "Ada", Role: RoleRider}
	if err := missing.Validate(); err == nil {
		t.Fatal("missing email must fail")
	}

	badRole := &User{Email: "a@b.com", Phone: "1", FullName: "Ada", Role: "ROOT"}
	if err := badRole.Validate(); err == nil {
		t.Fatal("invalid role must fail")
	}
}
