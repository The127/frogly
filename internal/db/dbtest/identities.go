package dbtest

import (
	"github.com/the127/frogly/internal/repositories"
)

// identitiesSuite checks identities: finding one by its login, and only one
// per login.
type identitiesSuite struct {
	base
}

// TestAnIdentityIsFoundByIssuerAndSubject checks that issuer and subject together find
// an identity, and nothing else does.
func (s *identitiesSuite) TestAnIdentityIsFoundByIssuerAndSubject() {
	// arrange
	user := s.savedUser("Karo")
	identity := s.savedIdentity(user)
	reading := s.database.NewContext().Identities()

	// act
	found, foundErr := reading.ByLogin(s.T().Context(), issuer, subject)
	otherIssuer, otherIssuerErr := reading.ByLogin(s.T().Context(), "http://elsewhere.example", subject)

	// assert
	s.Require().NoError(foundErr)
	s.Require().NotNil(found)
	s.Equal(identity.ID(), found.ID())
	s.Equal(user.ID(), found.UserID())

	s.Require().NoError(otherIssuerErr)
	s.Nil(otherIssuer)
}

// TestASecondIdentityForTheSameLoginIsAConcurrentModification checks that two
// identities can't share an issuer and subject.
func (s *identitiesSuite) TestASecondIdentityForTheSameLoginIsAConcurrentModification() {
	// arrange
	user := s.savedUser("Karo")
	s.savedIdentity(user)

	second, err := repositories.NewIdentity(user.ID(), issuer, subject)
	s.Require().NoError(err)

	racing := s.database.NewContext()
	racing.Identities().Insert(second)

	// act
	err = racing.SaveChanges(s.T().Context())

	// assert
	s.Require().ErrorIs(err, repositories.ErrConcurrentModification)
}
