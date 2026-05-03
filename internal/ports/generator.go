package ports

//go:generate mockery --name=UserRepository --output=../mocks --outpkg=mocks --filename=user_repository.go --structname=UserRepositoryMock --with-expecter=true
//go:generate mockery --name=PasswordHasher --output=../mocks --outpkg=mocks --filename=password_hasher.go --structname=PasswordHasherMock --with-expecter=true
//go:generate mockery --name=TokenManager --output=../mocks --outpkg=mocks --filename=token_manager.go --structname=TokenManagerMock --with-expecter=true
