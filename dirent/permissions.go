package dirent;

import (
    "os"
    "strconv"
    "strings"

    "github.com/pkg/errors"

    "github.com/eriq-augustine/elfs/identity"
)

const (
	_ = iota
	PERM_SU Permissions = 1 << (12 - iota)  // Set UID
	PERM_SG // Set GID
	PERM_ST // Sticky
	PERM_UR
	PERM_UW
	PERM_UX
	PERM_GR
	PERM_GW
	PERM_GX
	PERM_OR
	PERM_OW
	PERM_OX

    EMPTY_PERMISSIONS Permissions = 0
    // 0660
    DEFAULT_FILE_PERMISSIONS Permissions = EMPTY_PERMISSIONS | PERM_UR | PERM_UW | PERM_GR | PERM_GR
    // 0770
    DEFAULT_DIR_PERMISSIONS Permissions = DEFAULT_FILE_PERMISSIONS | PERM_UX | PERM_GX
)

type Permissions uint32;

// Check if this permissions contains some permissions.
func (this Permissions) Has(permissions Permissions) bool {
    return uint32(this) & uint32(permissions) != 0;
}

func (this Permissions) String() string {
    var builder strings.Builder;

    this.buildPermissionTriad(&builder, PERM_UR, PERM_UW, PERM_UX, PERM_SU, "s");
    this.buildPermissionTriad(&builder, PERM_GR, PERM_GW, PERM_GX, PERM_SG, "s");
    this.buildPermissionTriad(&builder, PERM_OR, PERM_OW, PERM_OX, PERM_ST, "t");

    return builder.String();
}

func (this Permissions) buildPermissionTriad(builder *strings.Builder,
        readPerm Permissions, writePerm Permissions, executePerm Permissions,
        specialPerm Permissions, specialCharacter string) {
    if (this.Has(readPerm)) {
        builder.WriteString("r");
    } else {
        builder.WriteString("-");
    }

    if (this.Has(writePerm)) {
        builder.WriteString("w");
    } else {
        builder.WriteString("-");
    }

    if (this.Has(executePerm) && this.Has(specialPerm)) {
        builder.WriteString(strings.ToLower(specialCharacter));
    } else if (this.Has(executePerm) && !this.Has(specialPerm)) {
        builder.WriteString("x");
    } else if (!this.Has(executePerm) && this.Has(specialPerm)) {
        builder.WriteString(strings.ToUpper(specialCharacter));
    } else {
        builder.WriteString("-");
    }
}

func PermissionsFromFileMode(mode os.FileMode) Permissions {
    var perms Permissions = EMPTY_PERMISSIONS;

    perms = checkFileModePerm(perms, mode, os.ModeSetuid, PERM_SU);
    perms = checkFileModePerm(perms, mode, os.ModeSetgid, PERM_SG);
    perms = checkFileModePerm(perms, mode, os.ModeSticky, PERM_ST);

    perms = checkFileModePerm(perms, mode, os.FileMode(0400), PERM_UR);
    perms = checkFileModePerm(perms, mode, os.FileMode(0200), PERM_UW);
    perms = checkFileModePerm(perms, mode, os.FileMode(0100), PERM_UX);

    perms = checkFileModePerm(perms, mode, os.FileMode(0040), PERM_GR);
    perms = checkFileModePerm(perms, mode, os.FileMode(0020), PERM_GW);
    perms = checkFileModePerm(perms, mode, os.FileMode(0010), PERM_GX);

    perms = checkFileModePerm(perms, mode, os.FileMode(0004), PERM_OR);
    perms = checkFileModePerm(perms, mode, os.FileMode(0002), PERM_OW);
    perms = checkFileModePerm(perms, mode, os.FileMode(0001), PERM_OX);

    return perms;
}

func checkFileModePerm(perms Permissions, mode os.FileMode, osPermission os.FileMode, elfsPermission Permissions) Permissions {
    if (mode & osPermission != 0) {
        perms |= elfsPermission;
    }

    return perms;
}

// Convert a string of the style: 775 in permissions.
// Takes four or three digit strings.
func PermissionsFromString(rawPerms string) (Permissions, error) {
    if (len(rawPerms) != 3 && len(rawPerms) != 4) {
        return EMPTY_PERMISSIONS, errors.Errorf("Error with permissions: '%s' -- Bad length (%d), expecting 3 or 4.", rawPerms, len(rawPerms));
    }

    var perms Permissions = EMPTY_PERMISSIONS;
    var err error;

    if (len(rawPerms) == 4) {
        perms, err = buildPermissionsTriadFromString(string(rawPerms[0]), perms, PERM_SU, PERM_SG, PERM_ST);
        if (err != nil) {
            return EMPTY_PERMISSIONS, errors.WithStack(err);
        }

        rawPerms = rawPerms[1:];
    }

    perms, err = buildPermissionsTriadFromString(string(rawPerms[0]), perms, PERM_UR, PERM_UW, PERM_UX);
    if (err != nil) {
        return EMPTY_PERMISSIONS, errors.WithStack(err);
    }

    perms, err = buildPermissionsTriadFromString(string(rawPerms[1]), perms, PERM_GR, PERM_GW, PERM_GX);
    if (err != nil) {
        return EMPTY_PERMISSIONS, errors.WithStack(err);
    }

    perms, err = buildPermissionsTriadFromString(string(rawPerms[2]), perms, PERM_OR, PERM_OW, PERM_OX);
    if (err != nil) {
        return EMPTY_PERMISSIONS, errors.WithStack(err);
    }

    return perms, nil;
}

func buildPermissionsTriadFromString(rawPerm string, perms Permissions,
        readPerm Permissions, writePerm Permissions, executePerm Permissions) (Permissions, error) {
    intPerm, err := strconv.Atoi(rawPerm);
    if (err != nil) {
        return EMPTY_PERMISSIONS, errors.Wrap(err, "Permission character must be int.");
    }

    if (intPerm >= 4) {
        intPerm -= 4;
        perms |= readPerm;
    }

    if (intPerm >= 2) {
        intPerm -= 2;
        perms |= writePerm;
    }

    if (intPerm >= 1) {
        intPerm -= 1;
        perms |= executePerm;
    }

    if (intPerm != 0) {
        return EMPTY_PERMISSIONS, errors.Errorf("Bad permission number (%s), expecting UNIX-style.", rawPerm);
    }

    return perms, nil;
}

// Can the specified user read the dirent.
func (this *Dirent) CanRead(user *identity.User, group *identity.Group) bool {
    if (this.Group != group.Id) {
        return false;
    }

    // Root can do anything.
    if (user.Id == identity.ROOT_USER_ID) {
        return true;
    }

    if (user.Id == this.Owner) {
        return this.Permissions.Has(PERM_UR);
    }

    if (group.HasMember(user.Id)) {
        return this.Permissions.Has(PERM_GR);
    }

    return this.Permissions.Has(PERM_OR);
}

// Can the specified user write to this dirent.
func (this *Dirent) CanWrite(user *identity.User, group *identity.Group) bool {
    if (this.Group != group.Id) {
        return false;
    }

    // Root can do anything.
    if (user.Id == identity.ROOT_USER_ID) {
        return true;
    }

    if (user.Id == this.Owner) {
        return this.Permissions.Has(PERM_UW);
    }

    if (group.HasMember(user.Id)) {
        return this.Permissions.Has(PERM_GW);
    }

    return this.Permissions.Has(PERM_OW);
}

// Can the specified user execute the dirent.
func (this *Dirent) CanExecute(user *identity.User, group *identity.Group) bool {
    if (this.Group != group.Id) {
        return false;
    }

    // Root can do anything.
    if (user.Id == identity.ROOT_USER_ID) {
        return true;
    }

    if (user.Id == this.Owner) {
        return this.Permissions.Has(PERM_UX);
    }

    if (group.HasMember(user.Id)) {
        return this.Permissions.Has(PERM_GX);
    }

    return this.Permissions.Has(PERM_OX);
}
