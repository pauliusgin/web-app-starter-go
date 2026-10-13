package main

import (
	"context"

	"github.com/pauliusgin/web_app_starter-be/internal/adapters/gateways/brevo"
	"github.com/pauliusgin/web_app_starter-be/internal/adapters/gateways/firebase"
	"github.com/pauliusgin/web_app_starter-be/internal/adapters/imaging/libvips"
	"github.com/pauliusgin/web_app_starter-be/internal/config"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/messaging/gateways/email"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/gateways/imaging"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/gateways/storage"
)

func buildEmailGateway(brevoConfig config.BrevoConfig) (email.Gateway, error) {
	emailGateway, err := brevo.NewEmailGateway(brevo.EmailGatewayParams{
		ApiKey:                 brevoConfig.ApiKey,
		SenderEmail:            brevoConfig.SenderEmail,
		VerificationTemplateID: brevoConfig.EmailVerificationTemplateID,
	})
	if err != nil {
		return nil, ErrEmailGatewayUnavailable{
			Variables: brevoConfig.EnvVariables(),
			Cause:     err,
		}
	}

	return emailGateway, nil
}

// fileStorageResource is the file storage the server runs with and how to release it on shutdown.
type fileStorageResource struct {
	FileStorage storage.FileStorage
	Close       func()
}

func buildFileStorage(firebaseConfig config.FirebaseConfig) (fileStorageResource, error) {
	gateway, err := firebase.NewStorageGateway(context.Background(), firebase.StorageGatewayParams{
		EncodedCredentials: firebaseConfig.EncodedCredentials,
		BucketName:         firebaseConfig.StorageBucket,
	})
	if err != nil {
		return fileStorageResource{}, ErrFileStorageUnavailable{
			Variables: firebaseConfig.EnvVariables(),
			Cause:     err,
		}
	}

	return fileStorageResource{
		FileStorage: gateway,
		Close:       func() { _ = gateway.Close() },
	}, nil
}

// imageProcessorResource is the image processor the server runs with and how to release it.
type imageProcessorResource struct {
	ImageProcessor imaging.ImageProcessor
	Close          func()
}

func buildImageProcessor() (imageProcessorResource, error) {
	processor, err := libvips.NewImageProcessor()
	if err != nil {
		return imageProcessorResource{}, ErrImageProcessorUnavailable{Cause: err}
	}

	return imageProcessorResource{
		ImageProcessor: processor,
		Close:          processor.Shutdown,
	}, nil
}
