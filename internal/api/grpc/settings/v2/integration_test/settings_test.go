//go:build integration

package settings_test

import (
	"context"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/muhlemmer/gu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/zitadel/zitadel/internal/integration"
	"github.com/zitadel/zitadel/pkg/grpc/idp"
	object_pb "github.com/zitadel/zitadel/pkg/grpc/object/v2"
	"github.com/zitadel/zitadel/pkg/grpc/settings/v2"
)

func TestServer_GetSecuritySettings(t *testing.T) {
	_, err := Client.SetSecuritySettings(AdminCTX, &settings.SetSecuritySettingsRequest{
		EmbeddedIframe: &settings.EmbeddedIframeSettings{
			Enabled:        true,
			AllowedOrigins: []string{"foo", "bar"},
		},
		EnableImpersonation: true,
	})
	require.NoError(t, err)

	tests := []struct {
		name    string
		ctx     context.Context
		want    *settings.GetSecuritySettingsResponse
		wantErr bool
	}{
		{
			name:    "permission error",
			ctx:     Instance.WithAuthorization(CTX, integration.UserTypeOrgOwner),
			wantErr: true,
		},
		{
			name: "success",
			ctx:  AdminCTX,
			want: &settings.GetSecuritySettingsResponse{
				Settings: &settings.SecuritySettings{
					EmbeddedIframe: &settings.EmbeddedIframeSettings{
						Enabled:        true,
						AllowedOrigins: []string{"foo", "bar"},
					},
					EnableImpersonation: true,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			retryDuration, tick := integration.WaitForAndTickWithMaxDuration(tt.ctx, time.Minute)
			assert.EventuallyWithT(t, func(ct *assert.CollectT) {
				resp, err := Client.GetSecuritySettings(tt.ctx, &settings.GetSecuritySettingsRequest{})
				if tt.wantErr {
					assert.Error(ct, err)
					return
				}
				if !assert.NoError(ct, err) {
					return
				}
				got, want := resp.GetSettings(), tt.want.GetSettings()
				assert.Equal(ct, want.GetEmbeddedIframe().GetEnabled(), got.GetEmbeddedIframe().GetEnabled(), "enable iframe embedding")
				assert.Equal(ct, want.GetEmbeddedIframe().GetAllowedOrigins(), got.GetEmbeddedIframe().GetAllowedOrigins(), "allowed origins")
				assert.Equal(ct, want.GetEnableImpersonation(), got.GetEnableImpersonation(), "enable impersonation")
			}, retryDuration, tick)
		})
	}
}

func TestServer_SetSecuritySettings(t *testing.T) {
	type args struct {
		ctx context.Context
		req *settings.SetSecuritySettingsRequest
	}
	tests := []struct {
		name    string
		args    args
		want    *settings.SetSecuritySettingsResponse
		wantErr bool
	}{
		{
			name: "permission error",
			args: args{
				ctx: Instance.WithAuthorization(CTX, integration.UserTypeOrgOwner),
				req: &settings.SetSecuritySettingsRequest{
					EmbeddedIframe: &settings.EmbeddedIframeSettings{
						Enabled:        true,
						AllowedOrigins: []string{"foo.com", "bar.com"},
					},
					EnableImpersonation: true,
				},
			},
			wantErr: true,
		},
		{
			name: "success allowed origins",
			args: args{
				ctx: AdminCTX,
				req: &settings.SetSecuritySettingsRequest{
					EmbeddedIframe: &settings.EmbeddedIframeSettings{
						AllowedOrigins: []string{"foo.com", "bar.com"},
					},
				},
			},
			want: &settings.SetSecuritySettingsResponse{
				Details: &object_pb.Details{
					ChangeDate:    timestamppb.Now(),
					ResourceOwner: Instance.ID(),
				},
			},
		},
		{
			name: "success enable iframe embedding",
			args: args{
				ctx: AdminCTX,
				req: &settings.SetSecuritySettingsRequest{
					EmbeddedIframe: &settings.EmbeddedIframeSettings{
						Enabled: true,
					},
				},
			},
			want: &settings.SetSecuritySettingsResponse{
				Details: &object_pb.Details{
					ChangeDate:    timestamppb.Now(),
					ResourceOwner: Instance.ID(),
				},
			},
		},
		{
			name: "success impersonation",
			args: args{
				ctx: AdminCTX,
				req: &settings.SetSecuritySettingsRequest{
					EnableImpersonation: true,
				},
			},
			want: &settings.SetSecuritySettingsResponse{
				Details: &object_pb.Details{
					ChangeDate:    timestamppb.Now(),
					ResourceOwner: Instance.ID(),
				},
			},
		},
		{
			name: "success all",
			args: args{
				ctx: AdminCTX,
				req: &settings.SetSecuritySettingsRequest{
					EmbeddedIframe: &settings.EmbeddedIframeSettings{
						Enabled:        true,
						AllowedOrigins: []string{"foo.com", "bar.com"},
					},
					EnableImpersonation: true,
				},
			},
			want: &settings.SetSecuritySettingsResponse{
				Details: &object_pb.Details{
					ChangeDate:    timestamppb.Now(),
					ResourceOwner: Instance.ID(),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Client.SetSecuritySettings(tt.args.ctx, tt.args.req)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			integration.AssertDetails(t, tt.want, got)
		})
	}
}

func TestServer_GetActiveIdentityProviders(t *testing.T) {
	instance := integration.NewInstance(CTX)
	isolatedIAMOwnerCTX := instance.WithAuthorization(CTX, integration.UserTypeIAMOwner)

	instance.AddGenericOAuthProvider(isolatedIAMOwnerCTX, gofakeit.AppName()) // inactive
	idpActiveName := gofakeit.AppName()
	idpActiveResp := instance.AddGenericOAuthProvider(isolatedIAMOwnerCTX, idpActiveName)
	instance.AddProviderToDefaultLoginPolicy(isolatedIAMOwnerCTX, idpActiveResp.GetId())
	idpLinkingDisallowedName := gofakeit.AppName()
	idpLinkingDisallowedResp := instance.AddGenericOAuthProviderWithOptions(isolatedIAMOwnerCTX, idpLinkingDisallowedName, false, true, true, idp.AutoLinkingOption_AUTO_LINKING_OPTION_USERNAME)
	instance.AddProviderToDefaultLoginPolicy(isolatedIAMOwnerCTX, idpLinkingDisallowedResp.GetId())
	idpCreationDisallowedName := gofakeit.AppName()
	idpCreationDisallowedResp := instance.AddGenericOAuthProviderWithOptions(isolatedIAMOwnerCTX, idpCreationDisallowedName, true, false, true, idp.AutoLinkingOption_AUTO_LINKING_OPTION_USERNAME)
	instance.AddProviderToDefaultLoginPolicy(isolatedIAMOwnerCTX, idpCreationDisallowedResp.GetId())
	idpNoAutoCreationName := gofakeit.AppName()
	idpNoAutoCreationResp := instance.AddGenericOAuthProviderWithOptions(isolatedIAMOwnerCTX, idpNoAutoCreationName, true, true, false, idp.AutoLinkingOption_AUTO_LINKING_OPTION_USERNAME)
	instance.AddProviderToDefaultLoginPolicy(isolatedIAMOwnerCTX, idpNoAutoCreationResp.GetId())
	idpNoAutoLinkingName := gofakeit.AppName()
	idpNoAutoLinkingResp := instance.AddGenericOAuthProviderWithOptions(isolatedIAMOwnerCTX, idpNoAutoLinkingName, true, true, true, idp.AutoLinkingOption_AUTO_LINKING_OPTION_UNSPECIFIED)
	instance.AddProviderToDefaultLoginPolicy(isolatedIAMOwnerCTX, idpNoAutoLinkingResp.GetId())
	type args struct {
		ctx context.Context
		req *settings.GetActiveIdentityProvidersRequest
	}
	tests := []struct {
		name    string
		args    args
		want    *settings.GetActiveIdentityProvidersResponse
		wantErr bool
	}{
		{
			name: "permission error",
			args: args{
				ctx: instance.WithAuthorization(CTX, integration.UserTypeLogin),
				req: &settings.GetActiveIdentityProvidersRequest{},
			},
			wantErr: true,
		},
		{
			name: "success, all",
			args: args{
				ctx: isolatedIAMOwnerCTX,
				req: &settings.GetActiveIdentityProvidersRequest{},
			},
			want: &settings.GetActiveIdentityProvidersResponse{
				Details: &object_pb.ListDetails{
					TotalResult: 5,
					Timestamp:   timestamppb.Now(),
				},
				IdentityProviders: []*settings.IdentityProvider{
					{
						Id:   idpActiveResp.GetId(),
						Name: idpActiveName,
						Type: settings.IdentityProviderType_IDENTITY_PROVIDER_TYPE_OAUTH,
					},
					{
						Id:   idpLinkingDisallowedResp.GetId(),
						Name: idpLinkingDisallowedName,
						Type: settings.IdentityProviderType_IDENTITY_PROVIDER_TYPE_OAUTH,
					},
					{
						Id:   idpCreationDisallowedResp.GetId(),
						Name: idpCreationDisallowedName,
						Type: settings.IdentityProviderType_IDENTITY_PROVIDER_TYPE_OAUTH,
					},
					{
						Id:   idpNoAutoCreationResp.GetId(),
						Name: idpNoAutoCreationName,
						Type: settings.IdentityProviderType_IDENTITY_PROVIDER_TYPE_OAUTH,
					},
					{
						Id:   idpNoAutoLinkingResp.GetId(),
						Name: idpNoAutoLinkingName,
						Type: settings.IdentityProviderType_IDENTITY_PROVIDER_TYPE_OAUTH,
					},
				},
			},
		},
		{
			name: "success, exclude linking disallowed",
			args: args{
				ctx: isolatedIAMOwnerCTX,
				req: &settings.GetActiveIdentityProvidersRequest{
					LinkingAllowed: gu.Ptr(true),
				},
			},
			want: &settings.GetActiveIdentityProvidersResponse{
				Details: &object_pb.ListDetails{
					TotalResult: 4,
					Timestamp:   timestamppb.Now(),
				},
				IdentityProviders: []*settings.IdentityProvider{
					{
						Id:   idpActiveResp.GetId(),
						Name: idpActiveName,
						Type: settings.IdentityProviderType_IDENTITY_PROVIDER_TYPE_OAUTH,
					},
					{
						Id:   idpCreationDisallowedResp.GetId(),
						Name: idpCreationDisallowedName,
						Type: settings.IdentityProviderType_IDENTITY_PROVIDER_TYPE_OAUTH,
					},
					{
						Id:   idpNoAutoCreationResp.GetId(),
						Name: idpNoAutoCreationName,
						Type: settings.IdentityProviderType_IDENTITY_PROVIDER_TYPE_OAUTH,
					},
					{
						Id:   idpNoAutoLinkingResp.GetId(),
						Name: idpNoAutoLinkingName,
						Type: settings.IdentityProviderType_IDENTITY_PROVIDER_TYPE_OAUTH,
					},
				},
			},
		},
		{
			name: "success, exclude creation disallowed",
			args: args{
				ctx: isolatedIAMOwnerCTX,
				req: &settings.GetActiveIdentityProvidersRequest{
					CreationAllowed: gu.Ptr(true),
				},
			},
			want: &settings.GetActiveIdentityProvidersResponse{
				Details: &object_pb.ListDetails{
					TotalResult: 4,
					Timestamp:   timestamppb.Now(),
				},
				IdentityProviders: []*settings.IdentityProvider{
					{
						Id:   idpActiveResp.GetId(),
						Name: idpActiveName,
						Type: settings.IdentityProviderType_IDENTITY_PROVIDER_TYPE_OAUTH,
					},
					{
						Id:   idpLinkingDisallowedResp.GetId(),
						Name: idpLinkingDisallowedName,
						Type: settings.IdentityProviderType_IDENTITY_PROVIDER_TYPE_OAUTH,
					},
					{
						Id:   idpNoAutoCreationResp.GetId(),
						Name: idpNoAutoCreationName,
						Type: settings.IdentityProviderType_IDENTITY_PROVIDER_TYPE_OAUTH,
					},
					{
						Id:   idpNoAutoLinkingResp.GetId(),
						Name: idpNoAutoLinkingName,
						Type: settings.IdentityProviderType_IDENTITY_PROVIDER_TYPE_OAUTH,
					},
				},
			},
		},
		{
			name: "success, auto creation",
			args: args{
				ctx: isolatedIAMOwnerCTX,
				req: &settings.GetActiveIdentityProvidersRequest{
					AutoCreation: gu.Ptr(true),
				},
			},
			want: &settings.GetActiveIdentityProvidersResponse{
				Details: &object_pb.ListDetails{
					TotalResult: 4,
					Timestamp:   timestamppb.Now(),
				},
				IdentityProviders: []*settings.IdentityProvider{
					{
						Id:   idpActiveResp.GetId(),
						Name: idpActiveName,
						Type: settings.IdentityProviderType_IDENTITY_PROVIDER_TYPE_OAUTH,
					},
					{
						Id:   idpLinkingDisallowedResp.GetId(),
						Name: idpLinkingDisallowedName,
						Type: settings.IdentityProviderType_IDENTITY_PROVIDER_TYPE_OAUTH,
					},
					{
						Id:   idpCreationDisallowedResp.GetId(),
						Name: idpCreationDisallowedName,
						Type: settings.IdentityProviderType_IDENTITY_PROVIDER_TYPE_OAUTH,
					},
					{
						Id:   idpNoAutoLinkingResp.GetId(),
						Name: idpNoAutoLinkingName,
						Type: settings.IdentityProviderType_IDENTITY_PROVIDER_TYPE_OAUTH,
					},
				},
			},
		}, {
			name: "success, auto linking",
			args: args{
				ctx: isolatedIAMOwnerCTX,
				req: &settings.GetActiveIdentityProvidersRequest{
					AutoLinking: gu.Ptr(true),
				},
			},
			want: &settings.GetActiveIdentityProvidersResponse{
				Details: &object_pb.ListDetails{
					TotalResult: 4,
					Timestamp:   timestamppb.Now(),
				},
				IdentityProviders: []*settings.IdentityProvider{
					{
						Id:   idpActiveResp.GetId(),
						Name: idpActiveName,
						Type: settings.IdentityProviderType_IDENTITY_PROVIDER_TYPE_OAUTH,
					},
					{
						Id:   idpLinkingDisallowedResp.GetId(),
						Name: idpLinkingDisallowedName,
						Type: settings.IdentityProviderType_IDENTITY_PROVIDER_TYPE_OAUTH,
					},
					{
						Id:   idpCreationDisallowedResp.GetId(),
						Name: idpCreationDisallowedName,
						Type: settings.IdentityProviderType_IDENTITY_PROVIDER_TYPE_OAUTH,
					},
					{
						Id:   idpNoAutoCreationResp.GetId(),
						Name: idpNoAutoCreationName,
						Type: settings.IdentityProviderType_IDENTITY_PROVIDER_TYPE_OAUTH,
					},
				},
			},
		},
		{
			name: "success, exclude all",
			args: args{
				ctx: isolatedIAMOwnerCTX,
				req: &settings.GetActiveIdentityProvidersRequest{
					LinkingAllowed:  gu.Ptr(true),
					CreationAllowed: gu.Ptr(true),
					AutoCreation:    gu.Ptr(true),
					AutoLinking:     gu.Ptr(true),
				},
			},
			want: &settings.GetActiveIdentityProvidersResponse{
				Details: &object_pb.ListDetails{
					TotalResult: 1,
					Timestamp:   timestamppb.Now(),
				},
				IdentityProviders: []*settings.IdentityProvider{
					{
						Id:   idpActiveResp.GetId(),
						Name: idpActiveName,
						Type: settings.IdentityProviderType_IDENTITY_PROVIDER_TYPE_OAUTH,
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			retryDuration, tick := integration.WaitForAndTickWithMaxDuration(tt.args.ctx, time.Minute)
			assert.EventuallyWithT(t, func(ct *assert.CollectT) {
				got, err := instance.Client.SettingsV2.GetActiveIdentityProviders(tt.args.ctx, tt.args.req)
				if tt.wantErr {
					assert.Error(ct, err)
					return
				}
				if !assert.NoError(ct, err) {
					return
				}
				for i, result := range tt.want.GetIdentityProviders() {
					assert.EqualExportedValues(ct, result, got.GetIdentityProviders()[i])
				}
				integration.AssertListDetails(ct, tt.want, got)
			}, retryDuration, tick)
		})
	}
}
