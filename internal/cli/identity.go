package cli

import (
	"context"
	"errors"
	"fmt"
	"io"

	gflashduty "github.com/flashcatcloud/go-flashduty"
)

type identityResult struct {
	AccountName string `json:"account_name"`
	MemberName  string `json:"member_name,omitempty"`
	Email       string `json:"email,omitempty"`
}

// resolveIdentity fetches the caller's identity, preferring member-level detail
// (which carries the member name) and falling back to account-level info when
// the app key is account-scoped rather than tied to a member.
func resolveIdentity(ctx context.Context, client *gflashduty.Client) (*identityResult, error) {
	member, _, memberErr := client.Members.MemberInfo(ctx)
	if memberErr == nil {
		return &identityResult{
			AccountName: member.AccountName,
			MemberName:  member.MemberName,
			Email:       member.Email,
		}, nil
	}

	account, _, accountErr := client.Account.Info(ctx)
	if accountErr == nil {
		return &identityResult{
			AccountName: account.AccountName,
			Email:       account.Email,
		}, nil
	}

	return nil, fmt.Errorf("authentication failed: %w", errors.Join(memberErr, accountErr))
}

func printIdentity(w io.Writer, id *identityResult) {
	_, _ = fmt.Fprintf(w, "  Account:  %s\n", id.AccountName)
	if id.MemberName != "" {
		_, _ = fmt.Fprintf(w, "  Member:   %s\n", id.MemberName)
	}
	if id.Email != "" {
		_, _ = fmt.Fprintf(w, "  Email:    %s\n", id.Email)
	}
}
