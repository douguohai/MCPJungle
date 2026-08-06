package permission

import (
	"sort"
	"testing"

	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/pkg/testhelpers"
	"github.com/mcpjungle/mcpjungle/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createServerWithSlug inserts an McpServer with an explicit Slug to avoid
// unique-index collisions (the test helper CreateTestMcpServer leaves Slug
// empty, which breaks when the table has a UNIQUE on slug).
func createServerWithSlug(t *testing.T, setup *testhelpers.TestDBSetup, name string) *model.McpServer {
	t.Helper()
	srv, err := model.NewStreamableHTTPServer(name, name, "http://localhost:1", "", nil, types.SessionModeStateless)
	require.NoError(t, err)
	srv.Slug = name
	require.NoError(t, setup.DB.Create(srv).Error)
	return srv
}

// setupPermissionTest creates a fresh test DB with two regular users and two
// MCP servers, returning the service and all IDs needed by test cases.
func setupPermissionTest(t *testing.T) (*Service, *testhelpers.TestDBSetup, *model.User, *model.User, *model.McpServer, *model.McpServer) {
	t.Helper()

	setup := testhelpers.SetupTestDB(t)
	svc := NewService(setup.DB)

	user1 := setup.CreateTestUser("alice", types.UserRoleMember)
	user2 := setup.CreateTestUser("bob", types.UserRoleMember)

	srv1 := createServerWithSlug(t, setup, "server-a")
	srv2 := createServerWithSlug(t, setup, "server-b")

	return svc, setup, user1, user2, srv1, srv2
}

// --- CRUD tests -----------------------------------------------------------

func TestCreateGroup(t *testing.T) {
	svc, _, user1, _, _, _ := setupPermissionTest(t)

	g, err := svc.CreateGroup("dev-team", "Dev Team", "Engineering team", user1.ID)
	require.NoError(t, err)
	assert.NotZero(t, g.ID)
	assert.Equal(t, "dev-team", g.Name)
	assert.Equal(t, "Dev Team", g.DisplayName)
	assert.Equal(t, "Engineering team", g.Description)
	assert.Equal(t, model.PermissionGroupStatusActive, g.Status)
	assert.Equal(t, user1.ID, g.CreatedByID)
}

func TestGetGroup(t *testing.T) {
	svc, _, user1, _, _, _ := setupPermissionTest(t)

	created, err := svc.CreateGroup("ops", "Ops", "", user1.ID)
	require.NoError(t, err)

	got, err := svc.GetGroup(created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, got.ID)
	assert.Equal(t, "ops", got.Name)
}

func TestGetGroup_NotFound(t *testing.T) {
	svc, _, _, _, _, _ := setupPermissionTest(t)

	_, err := svc.GetGroup(999999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestListGroups(t *testing.T) {
	svc, _, user1, _, _, _ := setupPermissionTest(t)

	_, err := svc.CreateGroup("g1", "G1", "", user1.ID)
	require.NoError(t, err)
	_, err = svc.CreateGroup("g2", "G2", "", user1.ID)
	require.NoError(t, err)

	groups, err := svc.ListGroups()
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(groups), 2)

	names := make([]string, 0, len(groups))
	for _, g := range groups {
		names = append(names, g.Name)
	}
	sort.Strings(names)
	assert.Contains(t, names, "g1")
	assert.Contains(t, names, "g2")
}

func TestUpdateGroup(t *testing.T) {
	svc, _, user1, _, _, _ := setupPermissionTest(t)

	g, err := svc.CreateGroup("eng", "Engineering", "old desc", user1.ID)
	require.NoError(t, err)

	err = svc.UpdateGroup(g.ID, "Engineering Team", "new desc")
	require.NoError(t, err)

	got, err := svc.GetGroup(g.ID)
	require.NoError(t, err)
	assert.Equal(t, "Engineering Team", got.DisplayName)
	assert.Equal(t, "new desc", got.Description)
}

func TestDisableGroup(t *testing.T) {
	svc, _, user1, _, _, _ := setupPermissionTest(t)

	g, err := svc.CreateGroup("temp", "Temporary", "", user1.ID)
	require.NoError(t, err)
	assert.Equal(t, model.PermissionGroupStatusActive, g.Status)

	err = svc.DisableGroup(g.ID)
	require.NoError(t, err)

	got, err := svc.GetGroup(g.ID)
	require.NoError(t, err)
	assert.Equal(t, model.PermissionGroupStatusDisabled, got.Status)
}

// --- Membership tests ------------------------------------------------------

func TestAddMember(t *testing.T) {
	svc, _, user1, user2, _, _ := setupPermissionTest(t)

	g, err := svc.CreateGroup("team", "Team", "", user1.ID)
	require.NoError(t, err)

	err = svc.AddMember(g.ID, user2.ID, user1.ID)
	require.NoError(t, err)

	members, err := svc.ListMembers(g.ID)
	require.NoError(t, err)
	assert.Len(t, members, 1)
	assert.Equal(t, user2.ID, members[0].UserID)
}

func TestAddMember_Idempotent(t *testing.T) {
	svc, _, user1, user2, _, _ := setupPermissionTest(t)

	g, err := svc.CreateGroup("team", "Team", "", user1.ID)
	require.NoError(t, err)

	err = svc.AddMember(g.ID, user2.ID, user1.ID)
	require.NoError(t, err)
	err = svc.AddMember(g.ID, user2.ID, user1.ID) // duplicate
	require.NoError(t, err)

	members, err := svc.ListMembers(g.ID)
	require.NoError(t, err)
	assert.Len(t, members, 1)
}

func TestRemoveMember(t *testing.T) {
	svc, _, user1, user2, _, _ := setupPermissionTest(t)

	g, err := svc.CreateGroup("team", "Team", "", user1.ID)
	require.NoError(t, err)
	require.NoError(t, svc.AddMember(g.ID, user2.ID, user1.ID))

	err = svc.RemoveMember(g.ID, user2.ID)
	require.NoError(t, err)

	members, err := svc.ListMembers(g.ID)
	require.NoError(t, err)
	assert.Len(t, members, 0)
}

func TestRemoveMember_NotFound(t *testing.T) {
	svc, _, user1, _, _, _ := setupPermissionTest(t)

	g, err := svc.CreateGroup("empty", "Empty", "", user1.ID)
	require.NoError(t, err)

	err = svc.RemoveMember(g.ID, 999999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// --- Service grant tests ---------------------------------------------------

func TestAddService(t *testing.T) {
	svc, _, user1, _, srv1, _ := setupPermissionTest(t)

	g, err := svc.CreateGroup("team", "Team", "", user1.ID)
	require.NoError(t, err)

	err = svc.AddService(g.ID, srv1.ID, user1.ID)
	require.NoError(t, err)

	services, err := svc.ListServices(g.ID)
	require.NoError(t, err)
	assert.Len(t, services, 1)
	assert.Equal(t, srv1.ID, services[0].McpServerID)
}

func TestAddService_Idempotent(t *testing.T) {
	svc, _, user1, _, srv1, _ := setupPermissionTest(t)

	g, err := svc.CreateGroup("team", "Team", "", user1.ID)
	require.NoError(t, err)

	require.NoError(t, svc.AddService(g.ID, srv1.ID, user1.ID))
	require.NoError(t, svc.AddService(g.ID, srv1.ID, user1.ID)) // duplicate

	services, err := svc.ListServices(g.ID)
	require.NoError(t, err)
	assert.Len(t, services, 1)
}

func TestRemoveService(t *testing.T) {
	svc, _, user1, _, srv1, _ := setupPermissionTest(t)

	g, err := svc.CreateGroup("team", "Team", "", user1.ID)
	require.NoError(t, err)
	require.NoError(t, svc.AddService(g.ID, srv1.ID, user1.ID))

	err = svc.RemoveService(g.ID, srv1.ID)
	require.NoError(t, err)

	services, err := svc.ListServices(g.ID)
	require.NoError(t, err)
	assert.Len(t, services, 0)
}

// --- UserEffectiveServices tests ------------------------------------------

func TestUserEffectiveServices_UnionAcrossGroups(t *testing.T) {
	// User belongs to 2 active groups with different service grants -> union.
	svc, _, user1, user2, srv1, srv2 := setupPermissionTest(t)

	g1, err := svc.CreateGroup("alpha", "Alpha", "", user1.ID)
	require.NoError(t, err)
	g2, err := svc.CreateGroup("beta", "Beta", "", user1.ID)
	require.NoError(t, err)

	require.NoError(t, svc.AddMember(g1.ID, user2.ID, user1.ID))
	require.NoError(t, svc.AddMember(g2.ID, user2.ID, user1.ID))
	require.NoError(t, svc.AddService(g1.ID, srv1.ID, user1.ID))
	require.NoError(t, svc.AddService(g2.ID, srv2.ID, user1.ID))

	ids, err := svc.UserEffectiveServices(user2.ID)
	require.NoError(t, err)
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	assert.Equal(t, []uint{srv1.ID, srv2.ID}, ids)
}

func TestUserEffectiveServices_DisabledGroupExcluded(t *testing.T) {
	svc, _, user1, user2, srv1, _ := setupPermissionTest(t)

	g, err := svc.CreateGroup("disabled-group", "Disabled", "", user1.ID)
	require.NoError(t, err)
	require.NoError(t, svc.AddMember(g.ID, user2.ID, user1.ID))
	require.NoError(t, svc.AddService(g.ID, srv1.ID, user1.ID))

	require.NoError(t, svc.DisableGroup(g.ID))

	ids, err := svc.UserEffectiveServices(user2.ID)
	require.NoError(t, err)
	assert.Len(t, ids, 0)
}

func TestUserEffectiveServices_NoGroups(t *testing.T) {
	svc, _, _, user2, _, _ := setupPermissionTest(t)

	ids, err := svc.UserEffectiveServices(user2.ID)
	require.NoError(t, err)
	assert.Len(t, ids, 0)
}

func TestUserEffectiveServices_MultiUserIsolation(t *testing.T) {
	// user1 in group with srv1, user2 in group with srv2 -> isolated.
	svc, _, user1, user2, srv1, srv2 := setupPermissionTest(t)

	g1, err := svc.CreateGroup("team-1", "Team 1", "", user1.ID)
	require.NoError(t, err)
	g2, err := svc.CreateGroup("team-2", "Team 2", "", user1.ID)
	require.NoError(t, err)

	require.NoError(t, svc.AddMember(g1.ID, user1.ID, user1.ID))
	require.NoError(t, svc.AddMember(g2.ID, user2.ID, user1.ID))
	require.NoError(t, svc.AddService(g1.ID, srv1.ID, user1.ID))
	require.NoError(t, svc.AddService(g2.ID, srv2.ID, user1.ID))

	ids1, err := svc.UserEffectiveServices(user1.ID)
	require.NoError(t, err)
	assert.Equal(t, []uint{srv1.ID}, ids1)

	ids2, err := svc.UserEffectiveServices(user2.ID)
	require.NoError(t, err)
	assert.Equal(t, []uint{srv2.ID}, ids2)
}

// --- Permission union boundary tests (task §5) ----------------------------

func TestUserEffectiveServices_EmptyGroup(t *testing.T) {
	// Group with no members and no service grants -> union is empty.
	svc, _, user1, _, _, _ := setupPermissionTest(t)

	_, err := svc.CreateGroup("empty-group", "Empty", "", user1.ID)
	require.NoError(t, err)

	ids, err := svc.UserEffectiveServices(user1.ID)
	require.NoError(t, err)
	assert.Len(t, ids, 0)
}

func TestUserEffectiveServices_DedupAcrossGroups(t *testing.T) {
	// User in 2 groups, both grant the same service -> dedup.
	svc, _, user1, user2, srv1, srv2 := setupPermissionTest(t)

	g1, err := svc.CreateGroup("a", "A", "", user1.ID)
	require.NoError(t, err)
	g2, err := svc.CreateGroup("b", "B", "", user1.ID)
	require.NoError(t, err)

	require.NoError(t, svc.AddMember(g1.ID, user2.ID, user1.ID))
	require.NoError(t, svc.AddMember(g2.ID, user2.ID, user1.ID))
	// Both groups grant srv1; only g1 grants srv2.
	require.NoError(t, svc.AddService(g1.ID, srv1.ID, user1.ID))
	require.NoError(t, svc.AddService(g1.ID, srv2.ID, user1.ID))
	require.NoError(t, svc.AddService(g2.ID, srv1.ID, user1.ID))

	ids, err := svc.UserEffectiveServices(user2.ID)
	require.NoError(t, err)
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	assert.Equal(t, []uint{srv1.ID, srv2.ID}, ids)
}

func TestUserEffectiveServices_UserNotInGroup(t *testing.T) {
	// Group grants a service, but user is not a member -> no access.
	svc, _, user1, user2, srv1, _ := setupPermissionTest(t)

	g, err := svc.CreateGroup("private", "Private", "", user1.ID)
	require.NoError(t, err)
	require.NoError(t, svc.AddService(g.ID, srv1.ID, user1.ID))
	// user2 is NOT added to the group.

	ids, err := svc.UserEffectiveServices(user2.ID)
	require.NoError(t, err)
	assert.Len(t, ids, 0)
}
