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
	) (challenge.GenerateResult, error) {
		return challenge.Generate(ctx, generateAuthChallengeDependencies, params)
	}

	// users
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

	verifyEmailDependencies := userusecases.VerifyEmailDependencies{
		UserWriteRepository:          usersrepo.NewUserWriteRepository(queries),
		EventDispatcher:              eventDispatcher,
		FindAuthChallengeByTokenHash: authrepo.NewAuthChallengeReadRepository(queries).FindByTokenHash,
		ConsumeAuthChallenge:         authrepo.NewAuthChallengeWriteRepository(queries).MarkUsed,
	}
	verifyEmail := func(ctx context.Context, params userusecases.VerifyEmailParams) error {
		return userusecases.VerifyEmail(ctx, verifyEmailDependencies, params)
	}

	deleteUserDependencies := userusecases.DeleteUserDependencies{
		UserWriteRepository: usersrepo.NewUserWriteRepository(queries),
		EventDispatcher:     eventDispatcher,
	}
	deleteUser := func(ctx context.Context, params userusecases.DeleteUserParams) error {
		return userusecases.DeleteUser(ctx, deleteUserDependencies, params)
	}

	requestEmailChangeDependencies := userusecases.RequestEmailChangeDependencies{
		UserWriteRepository:   usersrepo.NewUserWriteRepository(queries),
		GenerateAuthChallenge: generateAuthChallenge,
	}
	requestEmailChange := func(ctx context.Context, params userusecases.RequestEmailChangeParams) error {
		return userusecases.RequestEmailChange(ctx, requestEmailChangeDependencies, params)
	}

	changePasswordDependencies := userusecases.ChangePasswordDependencies{
		UserWriteRepository: usersrepo.NewUserWriteRepository(queries),
		EventDispatcher:     eventDispatcher,
	}
	changePassword := func(ctx context.Context, params userusecases.ChangePasswordParams) error {
		return userusecases.ChangePassword(ctx, changePasswordDependencies, params)
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

	// uploads
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

	// profiles
	createProfileDependencies := profileusecases.CreateProfileDependencies{
		ProfileWriteRepository: profilesrepo.NewProfileWriteRepository(queries),
		EventDispatcher:        eventDispatcher,
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

	// refresh_tokens
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

	return routes.RouterDependencies{
		Users:    userRouterDeps,
		Auth:     authRouterDeps,
		Uploads:  uploadRouterDeps,
		Profiles: profileRouterDeps,
	}, nil
}
