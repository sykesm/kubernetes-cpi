package cpi

type NotSupportedError struct{}

func (e NotSupportedError) Type() string  { return "Bosh::Clouds::NotSupported" }
func (e NotSupportedError) Error() string { return "Not supported" }

type NotImplementedError struct{}

func (e NotImplementedError) Type() string  { return "Bosh::Clouds::NotImplemented" }
func (e NotImplementedError) Error() string { return "Not implemented" }

type DiskNotAttachedError struct{}

func (e DiskNotAttachedError) Type() string  { return "Bosh::Clouds::DiskNotAttached" }
func (e DiskNotAttachedError) Error() string { return "Disk not attached" }
