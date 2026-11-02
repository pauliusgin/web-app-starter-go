package main

import (
	"context"
	"database/sql"
	"time"

	"github.com/pauliusgin/web_app_starter-be/internal/adapters/brokers/rabbitmq"
	authrepo "github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/auth/repositories"
	profilesrepo "github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/profiles/repositories"
	"github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/sqlc"
	uploadsrepo "github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/uploads/repositories"
	usersrepo "github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/users/repositories"
	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers"
	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers/messages"
	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers/profiles"
	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers/tokens"
	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers/uploads"
	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers/users"
	"github.com/pauliusgin/web_app_starter-be/internal/api/http/middleware"
	"github.com/pauliusgin/web_app_starter-be/internal/api/http/routes"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/challenge"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/googleidtoken"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/jwt"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/refreshtoken"
	profilequeries "github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/read/queries"
	profileusecases "github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/write/usecases"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/gateways/imaging"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/gateways/storage"
	uploadqueries "github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/read/queries"
	uploadusecases "github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/write/usecases"
	userqueries "github.com/pauliusgin/web_app_starter-be/internal/domains/users/read/queries"
	userusecases "github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/usecases"
)

type buildRouterDependenciesParams struct {
	Database       *sql.DB
	EventPublisher rabbitmq.EventPublisher
	JWTSecret      string
	AppEnv         string
	FileStorage    storage.FileStorage
	ImageProcessor imaging.ImageProcessor
	GoogleVerifier googleidtoken.Verifier
	GoogleClientID string
}

func buildRouterDependencies(params buildRouterDependenciesParams) (routes.RouterDependencies, error) {
	accessTokenIssuer, err := jwt.NewAccessTokenIssuer(jwt.NewAccessTokenIssuerParams{Secret: params.JWTSecret})
	if err != nil {
		return routes.RouterDependencies{}, err
	}

	queries := sqlc.New(params.Database)

	eventDispatcher := buildEventDispatcher(buildEventDispatcherParams{
		Queries:   queries,
		Publisher: params.EventPublisher.PublishEvent,
	})

	generateAuthChallengeDependencies := challenge.GenerateDependencies{
		WriteRepository: authrepo.NewAuthChallengeWriteRepository(queries),
		EventDispatcher: eventDispatcher,
	}
	generateAuthChallenge := func(
		ctx context.Context,
		params challenge.GenerateParams,
	) (challenge.GenerateResult, error) {
		return challenge.Generate(ctx, generateAuthChallengeDependencies, params)
	}

	ensureChallengeIssuanceAllowedDependencies := challenge.EnsureIssuanceAllowedDependencies{
		ReadRepository: authrepo.NewAuthChallengeReadRepository(queries),
		Now:            time.Now,
	}
	ensureChallengeIssuanceAllowed := func(ctx context.Context, params challenge.EnsureIssuanceAllowedParams) error {
		return challenge.EnsureIssuanceAllowed(ctx, ensureChallengeIssuanceAllowedDependencies, params)
	}

	signUpUserDependencies := userusecases.SignUpUserDependencies{
		UserWriteRepository:   usersrepo.NewUserWriteRepository(queries),
		EventDispatcher:       eventDispatcher,
		GenerateAuthChallenge: generateAuthChallenge,
	}
	signUpUser := func(ctx context.Context, params userusecases.SignUpUserParams) (userusecases.SignUpUserResult, error) {
		return userusecases.SignUpUser(ctx, signUpUserDependencies, params)
	}

	signInUserDependencies := userusecases.SignInUserDependencies{
		UserWriteRepository: usersrepo.NewUserWriteRepository(queries),
		EventDispatcher:     eventDispatcher,
	}
	signInUser := func(ctx context.Context, params userusecases.SignInUserParams) (userusecases.SignInUserResult, error) {
		return userusecases.SignInUser(ctx, signInUserDependencies, params)
	}

	// Deferred to the request, so wiring a nil verifier (as router_test does) cannot panic at build time.
	verifyGoogleIDToken := func(ctx context.Context, idToken string) (googleidtoken.Identity, error) {
		return params.GoogleVerifier.VerifyIDToken(ctx, idToken)
	}

	signUpUserGoogleDependencies := userusecases.SignUpUserGoogleDependencies{
		UserWriteRepository: usersrepo.NewUserWriteRepository(queries),
		EventDispatcher:     eventDispatcher,
		VerifyGoogleIDToken: verifyGoogleIDToken,
	}
	signUpUserGoogle := func(
		ctx context.Context,
		params userusecases.SignUpUserGoogleParams,
	) (userusecases.SignUpUserGoogleResult, error) {
		return userusecases.SignUpUserGoogle(ctx, signUpUserGoogleDependencies, params)
	}

	signInUserGoogleDependencies := userusecases.SignInUserGoogleDependencies{
		UserWriteRepository: usersrepo.NewUserWriteRepository(queries),
		EventDispatcher:     eventDispatcher,
		VerifyGoogleIDToken: verifyGoogleIDToken,
	}
	signInUserGoogle := func(
		ctx context.Context,
		params userusecases.SignInUserGoogleParams,
	) (userusecases.SignInUserGoogleResult, error) {
		return userusecases.SignInUserGoogle(ctx, signInUserGoogleDependencies, params)
	}

	verifyEmailDependencies := userusecases.VerifyEmailDependencies{
		UserWriteRepository:          usersrepo.NewUserWriteRepository(queries),
		EventDispatcher:              eventDispatcher,
		FindAuthChallengeByTokenHash: authrepo.NewAuthChallengeReadRepository(queries).FindByTokenHash,
		ConsumeAuthChallenge:         authrepo.NewAuthChallengeWriteRepository(queries).MarkUsed,
	}
	verifyEmail := func(ctx context.Context, params userusecases.VerifyEmailParams) error {
		return userusecases.VerifyEmail(ctx, verifyEmailDependencies, params)
	}

	getUserByIDDependencies := userqueries.GetUserByIDDependencies{
		UserReadRepository: usersrepo.NewUserReadRepository(queries),
	}
	getUserByID := func(
		ctx context.Context,
		params userqueries.GetUserByIDParams,
	) (userqueries.GetUserByIDResult, error) {
		return userqueries.GetUserByID(ctx, getUserByIDDependencies, params)
	}

	getGoogleSignInConfigDependencies := userqueries.GetGoogleSignInConfigDependencies{
		GoogleClientID: params.GoogleClientID,
	}
	getGoogleSignInConfig := func(ctx context.Context) (userqueries.GetGoogleSignInConfigResult, error) {
		return userqueries.GetGoogleSignInConfig(ctx, getGoogleSignInConfigDependencies)
	}

	deleteUserDependencies := userusecases.DeleteUserDependencies{
		UserWriteRepository: usersrepo.NewUserWriteRepository(queries),
		EventDispatcher:     eventDispatcher,
	}
	deleteUser := func(ctx context.Context, params userusecases.DeleteUserParams) error {
		return userusecases.DeleteUser(ctx, deleteUserDependencies, params)
	}

	requestEmailChangeDependencies := userusecases.RequestEmailChangeDependencies{
		UserWriteRepository:            usersrepo.NewUserWriteRepository(queries),
		EnsureChallengeIssuanceAllowed: ensureChallengeIssuanceAllowed,
		GenerateAuthChallenge:          generateAuthChallenge,
	}
	requestEmailChange := func(ctx context.Context, params userusecases.RequestEmailChangeParams) error {
		return userusecases.RequestEmailChange(ctx, requestEmailChangeDependencies, params)
	}

	resendEmailVerificationDependencies := userusecases.ResendEmailVerificationDependencies{
		UserWriteRepository:            usersrepo.NewUserWriteRepository(queries),
		EnsureChallengeIssuanceAllowed: ensureChallengeIssuanceAllowed,
		GenerateAuthChallenge:          generateAuthChallenge,
	}
	resendEmailVerification := func(ctx context.Context, params userusecases.ResendEmailVerificationParams) error {
		return userusecases.ResendEmailVerification(ctx, resendEmailVerificationDependencies, params)
	}

	requestPasswordResetDependencies := userusecases.RequestPasswordResetDependencies{
		UserWriteRepository:            usersrepo.NewUserWriteRepository(queries),
		EnsureChallengeIssuanceAllowed: ensureChallengeIssuanceAllowed,
		GenerateAuthChallenge:          generateAuthChallenge,
	}
	requestPasswordReset := func(ctx context.Context, params userusecases.RequestPasswordResetParams) error {
		return userusecases.RequestPasswordReset(ctx, requestPasswordResetDependencies, params)
	}

	deleteUserRefreshTokensDependencies := refreshtoken.DeleteUserRefreshTokensDependencies{
		WriteRepository: authrepo.NewRefreshTokenWriteRepository(queries),
	}
	deleteUserRefreshTokens := func(ctx context.Context, params refreshtoken.DeleteUserRefreshTokensParams) error {
		return refreshtoken.DeleteUserRefreshTokens(ctx, deleteUserRefreshTokensDependencies, params)
	}

	resetPasswordDependencies := userusecases.ResetPasswordDependencies{
		UserWriteRepository:          usersrepo.NewUserWriteRepository(queries),
		EventDispatcher:              eventDispatcher,
		FindAuthChallengeByTokenHash: authrepo.NewAuthChallengeReadRepository(queries).FindByTokenHash,
		ConsumeAuthChallenge:         authrepo.NewAuthChallengeWriteRepository(queries).MarkUsed,
		DeleteUserRefreshTokens:      deleteUserRefreshTokens,
	}
	resetPassword := func(ctx context.Context, params userusecases.ResetPasswordParams) error {
		return userusecases.ResetPassword(ctx, resetPasswordDependencies, params)
	}

	changePasswordDependencies := userusecases.ChangePasswordDependencies{
		UserWriteRepository: usersrepo.NewUserWriteRepository(queries),
		EventDispatcher:     eventDispatcher,
	}
	changePassword := func(ctx context.Context, params userusecases.ChangePasswordParams) error {
		return userusecases.ChangePassword(ctx, changePasswordDependencies, params)
	}

	addPasswordDependencies := userusecases.AddPasswordDependencies{
		UserWriteRepository: usersrepo.NewUserWriteRepository(queries),
		EventDispatcher:     eventDispatcher,
	}
	addPassword := func(ctx context.Context, params userusecases.AddPasswordParams) error {
		return userusecases.AddPassword(ctx, addPasswordDependencies, params)
	}

	addGoogleAccountDependencies := userusecases.AddGoogleAccountDependencies{
		UserWriteRepository: usersrepo.NewUserWriteRepository(queries),
		EventDispatcher:     eventDispatcher,
		VerifyGoogleIDToken: verifyGoogleIDToken,
	}
	addGoogleAccount := func(ctx context.Context, params userusecases.AddGoogleAccountParams) error {
		return userusecases.AddGoogleAccount(ctx, addGoogleAccountDependencies, params)
	}

	uploadFilesDependencies := uploadusecases.UploadFilesDependencies{
		UploadWriteRepository: uploadsrepo.NewUploadWriteRepository(queries),
		FileStorage:           params.FileStorage,
		ImageProcessor:        params.ImageProcessor,
		EventDispatcher:       eventDispatcher,
		AppEnv:                params.AppEnv,
	}
	uploadFiles := func(
		ctx context.Context,
		params uploadusecases.UploadFilesParams,
	) (uploadusecases.UploadFilesResult, error) {
		return uploadusecases.UploadFiles(ctx, uploadFilesDependencies, params)
	}

	getUploadsByIDsDependencies := uploadqueries.GetUploadsByIDsDependencies{
		UploadReadRepository: uploadsrepo.NewUploadReadRepository(queries),
	}
	getUploadsByIDs := func(
		ctx context.Context,
		params uploadqueries.GetUploadsByIDsParams,
	) (uploadqueries.GetUploadsByIDsResult, error) {
		return uploadqueries.GetUploadsByIDs(ctx, getUploadsByIDsDependencies, params)
	}

	createProfileDependencies := profileusecases.CreateProfileDependencies{
		ProfileWriteRepository: profilesrepo.NewProfileWriteRepository(queries),
		EventDispatcher:        eventDispatcher,
		AttachUpload:           uploadsrepo.NewUploadWriteRepository(queries).AttachUpload,
	}
	createProfile := func(
		ctx context.Context,
		params profileusecases.CreateProfileParams,
	) (profileusecases.CreateProfileResult, error) {
		return profileusecases.CreateProfile(ctx, createProfileDependencies, params)
	}

	updateProfileDependencies := profileusecases.UpdateProfileDependencies{
		ProfileWriteRepository: profilesrepo.NewProfileWriteRepository(queries),
		EventDispatcher:        eventDispatcher,
		AttachUpload:           uploadsrepo.NewUploadWriteRepository(queries).AttachUpload,
	}
	updateProfile := func(
		ctx context.Context,
		params profileusecases.UpdateProfileParams,
	) (profileusecases.UpdateProfileResult, error) {
		return profileusecases.UpdateProfile(ctx, updateProfileDependencies, params)
	}

	getProfileByIDDependencies := profilequeries.GetProfileByIDDependencies{
		ProfileReadRepository: profilesrepo.NewProfileReadRepository(queries),
	}
	getProfileByID := func(
		ctx context.Context,
		params profilequeries.GetProfileByIDParams,
	) (profilequeries.GetProfileByIDResult, error) {
		return profilequeries.GetProfileByID(ctx, getProfileByIDDependencies, params)
	}

	issueRefreshTokenDependencies := refreshtoken.IssueDependencies{
		WriteRepository: authrepo.NewRefreshTokenWriteRepository(queries),
	}
	issueRefreshToken := func(
		ctx context.Context,
		params refreshtoken.IssueParams,
	) (refreshtoken.IssueResult, error) {
		return refreshtoken.Issue(ctx, issueRefreshTokenDependencies, params)
	}

	rotateRefreshTokenDependencies := refreshtoken.RotateDependencies{
		ReadRepository:  authrepo.NewRefreshTokenReadRepository(queries),
		WriteRepository: authrepo.NewRefreshTokenWriteRepository(queries),
	}
	rotateRefreshToken := func(
		ctx context.Context,
		params refreshtoken.RotateParams,
	) (refreshtoken.RotateResult, error) {
		return refreshtoken.Rotate(ctx, rotateRefreshTokenDependencies, params)
	}

	revokeRefreshTokenDependencies := refreshtoken.RevokeDependencies{
		ReadRepository:  authrepo.NewRefreshTokenReadRepository(queries),
		WriteRepository: authrepo.NewRefreshTokenWriteRepository(queries),
	}
	revokeRefreshToken := func(
		ctx context.Context,
		params refreshtoken.RevokeParams,
	) error {
		return refreshtoken.Revoke(ctx, revokeRefreshTokenDependencies, params)
	}

	requireAuthenticatedUser := middleware.RequireAuthenticatedUserDependencies{
		ParseAccessToken: accessTokenIssuer.ParseAccessToken,
		FindUserByID:     usersrepo.NewUserWriteRepository(queries).FindUserByID,
	}

	userRouterDeps := routes.UserRoutesDependencies{
		SignUpUser: users.SignUpUserDependencies{
			SignUpUser:           signUpUser,
			GetUserReadModelByID: usersrepo.NewUserReadRepository(queries).GetUserReadModelByID,
			IssueRefreshToken:    issueRefreshToken,
			Jwt:                  accessTokenIssuer,
		},
		SignInUser: users.SignInUserDependencies{
			SignInUser:           signInUser,
			GetUserReadModelByID: usersrepo.NewUserReadRepository(queries).GetUserReadModelByID,
			IssueRefreshToken:    issueRefreshToken,
			Jwt:                  accessTokenIssuer,
		},
		SignUpUserGoogle: users.SignUpUserGoogleDependencies{
			SignUpUserGoogle:     signUpUserGoogle,
			GetUserReadModelByID: usersrepo.NewUserReadRepository(queries).GetUserReadModelByID,
			IssueRefreshToken:    issueRefreshToken,
			Jwt:                  accessTokenIssuer,
		},
		SignInUserGoogle: users.SignInUserGoogleDependencies{
			SignInUserGoogle:     signInUserGoogle,
			GetUserReadModelByID: usersrepo.NewUserReadRepository(queries).GetUserReadModelByID,
			IssueRefreshToken:    issueRefreshToken,
			Jwt:                  accessTokenIssuer,
		},
		GetGoogleSignInConfig: users.GetGoogleSignInConfigDependencies{
			GetGoogleSignInConfig: getGoogleSignInConfig,
		},
		VerifyEmail: users.VerifyEmailDependencies{
			VerifyEmail: verifyEmail,
		},
		GetUserByID: users.GetUserByIDDependencies{
			GetUserByID: getUserByID,
		},
		DeleteUser: users.DeleteUserDependencies{
			DeleteUser: deleteUser,
		},
		RequestEmailChange: users.RequestEmailChangeDependencies{
			RequestEmailChange: requestEmailChange,
		},
		ChangePassword: users.ChangePasswordDependencies{
			ChangePassword: changePassword,
		},
		AddPassword: users.AddPasswordDependencies{
			AddPassword: addPassword,
		},
		ResetPassword: users.ResetPasswordDependencies{
			ResetPassword: resetPassword,
		},
		AddGoogleAccount: users.AddGoogleAccountDependencies{
			AddGoogleAccount: addGoogleAccount,
		},
		RequireAuthenticatedUser: requireAuthenticatedUser,
	}

	authRouterDeps := routes.TokenRoutesDependencies{
		RotateRefreshToken: tokens.RotateRefreshTokenDependencies{
			RotateRefreshToken: rotateRefreshToken,
			FindUserByID:       usersrepo.NewUserWriteRepository(queries).FindUserByID,
			Jwt:                accessTokenIssuer,
		},
		RevokeRefreshToken: tokens.RevokeRefreshTokenDependencies{
			RevokeRefreshToken: revokeRefreshToken},
	}

	uploadRouterDeps := routes.UploadRoutesDependencies{
		UploadFiles: uploads.UploadFilesDependencies{
			UploadFiles:     uploadFiles,
			GetUploadsByIDs: getUploadsByIDs,
		},
		RequireAuthenticatedUser: requireAuthenticatedUser,
	}

	profileRouterDeps := routes.ProfileRoutesDependencies{
		CreateProfile: profiles.CreateProfileDependencies{
			CreateProfile:  createProfile,
			GetProfileByID: getProfileByID,
		},
		UpdateProfile: profiles.UpdateProfileDependencies{
			UpdateProfile:  updateProfile,
			GetProfileByID: getProfileByID,
		},
		RequireAuthenticatedUser: requireAuthenticatedUser,
	}

	messageRouterDeps := routes.MessageRoutesDependencies{
		ResendEmailVerification: messages.ResendEmailVerificationDependencies{
			ResendEmailVerification: resendEmailVerification,
		},
		SendPasswordResetEmail: messages.SendPasswordResetEmailDependencies{
			RequestPasswordReset: requestPasswordReset,
			Clock:                handlers.SystemResponseClock,
		},
		RequireAuthenticatedUser: requireAuthenticatedUser,
	}

	return routes.RouterDependencies{
		Users:    userRouterDeps,
		Auth:     authRouterDeps,
		Uploads:  uploadRouterDeps,
		Profiles: profileRouterDeps,
		Messages: messageRouterDeps,
	}, nil
}
