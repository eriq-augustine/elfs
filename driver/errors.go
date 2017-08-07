package driver;

// Driver-specific errors.

type PermissionsError struct {
   message string
}

func NewPermissionsError(message string) *PermissionsError {
   return &PermissionsError{message};
}

func (this *PermissionsError) Error() string {
   return "Permissions Error: " + this.message;
}

type IllegalOperationError struct {
   message string
}

func NewIllegalOperationError(message string) *IllegalOperationError {
   return &IllegalOperationError{message};
}

func (this *IllegalOperationError) Error() string {
   return "Illegal Operation Error: " + this.message;
}
