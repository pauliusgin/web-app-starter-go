package main

import (
	"context"
	"database/sql"

	"github.com/pauliusgin/web_app_starter-be/internal/adapters/brokers/rabbitmq"
	authrepo "github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/auth/repositories"
	profilesrepo "github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/profiles/repositories"
	"github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/sqlc"
	uploadsrepo "github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/uploads/repositories"
	usersrepo "github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/users/repositories"
	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers/profiles"
	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers/tokens"
	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers/uploads"
	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers/users"
	"github.com/pauliusgin/web_app_starter-be/internal/api/http/middleware"
	"github.com/pauliusgin/web_app_starter-be/internal/api/http/routes"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/challenge"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/jwt"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/refreshtoken"
	profileusecases "github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/usecases"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/gateways/imaging"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/gateways/storage"
	uploadusecases "github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/usecases"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/usecases"
)

type buildRouterDependenciesParams struct {
	Database       *sql.DB
	EventPublisher rabbitmq.EventPublisher
	JWTSecret      string
	AppEnv         string
	FileStorage    storage.FileStorage
	ImageProcessor imaging.ImageProcessor
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

	// auth_challenges
	generateAuthChallengeDependencies := challenge.GenerateDependencies{
		WriteRepository: authrepo.NewAuthChallengeWriteRepository(queries),
		EventDispatcher: eventDispatcher,
	}
	generateAuthChallenge := func(
		ctx context.Context,
		params challenge.GenerateParams,
	) (challenge.Generated, error) {
		return challenge.Generate(ctx, generateAuthChallengeDependencies, params)
	}

	// users
	signupUserDependencies := usecases.SignupUserDependencies{
		UserWriteRepository:   usersrepo.NewUserWriteRepository(queries),
		EventDispatcher:       eventDispatcher,
		GenerateAuthChallenge: generateAuthChallenge,
	}
	signupUser := func(ctx context.Context, params usecases.SignupUserParams) (usecases.SignupUserResult, error) {
		return usecases.SignupUser(ctx, signupUserDependencies, params)
	}

	signInUserDependencies := usecases.SignInUserDependencies{
		UserWriteRepository: usersrepo.NewUserWriteRepository(queries),
		EventDispatcher:     eventDispatcher,
	}
	signInUser := func(ctx context.Context, params usecases.SignInUserParams) (usecases.SignInUserResult, error) {
		return usecases.SignInUser(ctx, signInUserDependencies, params)
	}

	verifyEmailDependencies := usecases.VerifyEmailDependencies{
		UserWriteRepository:          usersrepo.NewUserWriteRepository(queries),
		EventDispatcher:              eventDispatcher,
		FindAuthChallengeByTokenHash: authrepo.NewAuthChallengeReadRepository(queries).FindByTokenHash,
		ConsumeAuthChallenge:         authrepo.NewAuthChallengeWriteRepository(queries).MarkUsed,
	}
	verifyEmail := func(ctx context.Context, params usecases.VerifyEmailParams) error {
		return usecases.VerifyEmail(ctx, verifyEmailDependencies, params)
	}

	deleteUserDependencies := usecases.DeleteUserDependencies{
		UserWriteRepository: usersrepo.NewUserWriteRepository(queries),
		EventDispatcher:     eventDispatcher,
	}
	deleteUser := func(ctx context.Context, params usecases.DeleteUserParams) error {
		return usecases.DeleteUser(ctx, deleteUserDependencies, params)
	}

	requestEmailChangeDependencies := usecases.RequestEmailChangeDependencies{
		UserWriteRepository:   usersrepo.NewUserWriteRepository(queries),
		GenerateAuthChallenge: generateAuthChallenge,
	}
	requestEmailChange := func(ctx context.Context, params usecases.RequestEmailChangeParams) error {
		return usecases.RequestEmailChange(ctx, requestEmailChangeDependencies, params)
	}

	changePasswordDependencies := usecases.ChangePasswordDependencies{
		UserWriteRepository: usersrepo.NewUserWriteRepository(queries),
		EventDispatcher:     eventDispatcher,
	}
	changePassword := func(ctx context.Context, params usecases.ChangePasswordParams) error {
		return usecases.ChangePassword(ctx, changePasswordDependencies, params)
	}

	// uploads
	uploadFilesDependencies := uploadusecases.UploadFilesDependencies{
		UploadRepository: uploadsrepo.NewUploadRepository(queries),
		FileStorage:      params.FileStorage,
		ImageProcessor:   params.ImageProcessor,
		EventDispatcher:  eventDispatcher,
		AppEnv:           params.AppEnv,
	}
	uploadFiles := func(
		ctx context.Context,
		params uploadusecases.UploadFilesParams,
	) (uploadusecases.UploadFilesResult, error) {
		return uploadusecases.UploadFiles(ctx, uploadFilesDependencies, params)
	}

	// profiles
	createProfileDependencies := profileusecases.CreateProfileDependencies{
		ProfileRepository: profilesrepo.NewProfileRepository(queries),
		EventDispatcher:   eventDispatcher,
	}
	createProfile := func(
		ctx context.Context,
		params profileusecases.CreateProfileParams,
	) (profileusecases.CreateProfileResult, error) {
		return profileusecases.CreateProfile(ctx, createProfileDependencies, params)
	}

	updateProfileDependencies := profileusecases.UpdateProfileDependencies{
		ProfileRepository: profilesrepo.NewProfileRepository(queries),
		EventDispatcher:   eventDispatcher,
	}
	updateProfile := func(
		ctx context.Context,
		params profileusecases.UpdateProfileParams,
	) (profileusecases.UpdateProfileResult, error) {
		return profileusecases.UpdateProfile(ctx, updateProfileDependencies, params)
	}

	// refresh_tokens
	issueRefreshTokenDependencies := refreshtoken.IssuingDependencies{
		WriteRepository: authrepo.NewRefreshTokenWriteRepository(queries),
	}
	refreshTokenIssuer := func(
		ctx context.Context,
		params refreshtoken.IssuingParams,
	) (refreshtoken.Issued, error) {
		return refreshtoken.Issue(ctx, issueRefreshTokenDependencies, params)
	}

	rotateRefreshTokenDependencies := refreshtoken.RotateDependencies{
		ReadRepository:  authrepo.NewRefreshTokenReadRepository(queries),
		WriteRepository: authrepo.NewRefreshTokenWriteRepository(queries),
	}
	refreshTokenRotator := func(
		ctx context.Context,
		params refreshtoken.RotateParams,
	) (refreshtoken.Rotated, error) {
		return refreshtoken.Rotate(ctx, rotateRefreshTokenDependencies, params)
	}

	revokeRefreshTokenDependencies := refreshtoken.RevokeDependencies{
		ReadRepository:  authrepo.NewRefreshTokenReadRepository(queries),
		WriteRepository: authrepo.NewRefreshTokenWriteRepository(queries),
	}
	refreshTokenRevoker := func(
		ctx context.Context,
		params refreshtoken.RevokeParams,
	) error {
		return refreshtoken.Revoke(ctx, revokeRefreshTokenDependencies, params)
	}

	requireAuthenticatedUser := middleware.RequireAuthenticatedUserDependencies{
		ParseAccessToken: accessTokenIssuer.ParseAccessToken,
		FindUserByID:     usersrepo.NewUserWriteRepository(queries).FindUserByID,
	}

	userRouterDeps := routes.UserRouterDependencies{
		SignUp: users.SignUpUserDependencies{
			SignupUser:        signupUser,
			IssueRefreshToken: refreshTokenIssuer,
			Jwt:               accessTokenIssuer,
		},
		SignIn: users.SignInUserDependencies{
			SignInUser:            signInUser,
			FindUserReadModelByID: usersrepo.NewUserReadRepository(queries).FindUserReadModelByID,
			IssueRefreshToken:     refreshTokenIssuer,
			Jwt:                   accessTokenIssuer,
		},
		VerifyEmail: users.VerifyEmailDependencies{
			VerifyEmail: verifyEmail,
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
		RequireAuthenticatedUser: requireAuthenticatedUser,
	}

	authRouterDeps := routes.AuthRouterDependencies{
		RotateRefreshToken: tokens.RotateDependencies{
			RotateRefreshToken: refreshTokenRotator,
			FindUserByID:       usersrepo.NewUserWriteRepository(queries).FindUserByID,
			Jwt:                accessTokenIssuer,
		},
		RevokeRefreshToken: tokens.RevokeRefreshTokenDependencies{
			RevokeRefreshToken: refreshTokenRevoker},
	}

	uploadRouterDeps := routes.UploadRouterDependencies{
		UploadFiles: uploads.UploadFilesDependencies{
			UploadFiles: uploadFiles,
		},
		RequireAuthenticatedUser: requireAuthenticatedUser,
	}

	profileRouterDeps := routes.ProfileRouterDependencies{
		CreateProfile: profiles.CreateProfileDependencies{
			CreateProfile: createProfile,
		},
		UpdateProfile: profiles.UpdateProfileDependencies{
			UpdateProfile: updateProfile,
		},
		RequireAuthenticatedUser: requireAuthenticatedUser,
	}

	return routes.RouterDependencies{
		Users:    userRouterDeps,
		Auth:     authRouterDeps,
		Uploads:  uploadRouterDeps,
		Profiles: profileRouterDeps,
	}, nil
}
