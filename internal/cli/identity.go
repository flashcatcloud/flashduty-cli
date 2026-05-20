package cli

import (
	"context"
	"errors"
	"fmt"
	"io"

	flashduty "github.com/flashcatcloud/flashduty-sdk"
)

type identityResult struct {
	AccountName string `json:"account_name"`
	MemberName  string `json:"member_name,omitempty"`
	Email       string `json:"email,omitempty"`
}

type identityClient interface {
	GetAccountInfo(ctx context.Context) (*flashduty.AccountInfo, error)
	GetMemberInfo(ctx context.Context) (*flashduty.MemberInfo, error)
}

func resolveIdentity(ctx context.Context, client identityClient) (*identityResult, error) {
	member, memberErr := client.GetMemberInfo(ctx)
	if memberErr == nil {
		return &identityResult{
			AccountName: member.AccountName,
			MemberName:  member.MemberName,
			Email:       member.Email,
		}, nil
	}

	account, accountErr := client.GetAccountInfo(ctx)
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
