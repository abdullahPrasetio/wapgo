package entity

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUser_TableName(t *testing.T) {
	assert.Equal(t, "users", User{}.TableName())
}

func TestUser_BeforeCreate_AssignsUUID_WhenNil(t *testing.T) {
	u := &User{}
	err := u.BeforeCreate(nil)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, u.ID)
}

func TestUser_BeforeCreate_PreservesExistingUUID(t *testing.T) {
	id := uuid.New()
	u := &User{ID: id}
	err := u.BeforeCreate(nil)
	require.NoError(t, err)
	assert.Equal(t, id, u.ID)
}

func TestUser_BeforeCreate_UniqueIDs(t *testing.T) {
	u1, u2 := &User{}, &User{}
	require.NoError(t, u1.BeforeCreate(nil))
	require.NoError(t, u2.BeforeCreate(nil))
	assert.NotEqual(t, u1.ID, u2.ID)
}
