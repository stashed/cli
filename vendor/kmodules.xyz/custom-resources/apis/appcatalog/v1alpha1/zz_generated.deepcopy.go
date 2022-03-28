//go:build !ignore_autogenerated
// +build !ignore_autogenerated

/*
Copyright AppsCode Inc. and Contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by deepcopy-gen. DO NOT EDIT.

package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AddKeyTransform) DeepCopyInto(out *AddKeyTransform) {
	*out = *in
	if in.Value != nil {
		in, out := &in.Value, &out.Value
		*out = make([]byte, len(*in))
		copy(*out, *in)
	}
	if in.StringValue != nil {
		in, out := &in.StringValue, &out.StringValue
		*out = new(string)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AddKeyTransform.
func (in *AddKeyTransform) DeepCopy() *AddKeyTransform {
	if in == nil {
		return nil
	}
	out := new(AddKeyTransform)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AddKeysFromTransform) DeepCopyInto(out *AddKeysFromTransform) {
	*out = *in
	if in.SecretRef != nil {
		in, out := &in.SecretRef, &out.SecretRef
		*out = new(v1.LocalObjectReference)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AddKeysFromTransform.
func (in *AddKeysFromTransform) DeepCopy() *AddKeysFromTransform {
	if in == nil {
		return nil
	}
	out := new(AddKeysFromTransform)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AppBinding) DeepCopyInto(out *AppBinding) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AppBinding.
func (in *AppBinding) DeepCopy() *AppBinding {
	if in == nil {
		return nil
	}
	out := new(AppBinding)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *AppBinding) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AppBindingList) DeepCopyInto(out *AppBindingList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]AppBinding, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AppBindingList.
func (in *AppBindingList) DeepCopy() *AppBindingList {
	if in == nil {
		return nil
	}
	out := new(AppBindingList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *AppBindingList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AppBindingSpec) DeepCopyInto(out *AppBindingSpec) {
	*out = *in
	in.ClientConfig.DeepCopyInto(&out.ClientConfig)
	if in.Secret != nil {
		in, out := &in.Secret, &out.Secret
		*out = new(v1.LocalObjectReference)
		**out = **in
	}
	if in.SecretTransforms != nil {
		in, out := &in.SecretTransforms, &out.SecretTransforms
		*out = make([]SecretTransform, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Parameters != nil {
		in, out := &in.Parameters, &out.Parameters
		*out = new(runtime.RawExtension)
		(*in).DeepCopyInto(*out)
	}
	if in.TLSSecret != nil {
		in, out := &in.TLSSecret, &out.TLSSecret
		*out = new(v1.LocalObjectReference)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AppBindingSpec.
func (in *AppBindingSpec) DeepCopy() *AppBindingSpec {
	if in == nil {
		return nil
	}
	out := new(AppBindingSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AppReference) DeepCopyInto(out *AppReference) {
	*out = *in
	if in.Parameters != nil {
		in, out := &in.Parameters, &out.Parameters
		*out = new(runtime.RawExtension)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AppReference.
func (in *AppReference) DeepCopy() *AppReference {
	if in == nil {
		return nil
	}
	out := new(AppReference)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClientConfig) DeepCopyInto(out *ClientConfig) {
	*out = *in
	if in.URL != nil {
		in, out := &in.URL, &out.URL
		*out = new(string)
		**out = **in
	}
	if in.Service != nil {
		in, out := &in.Service, &out.Service
		*out = new(ServiceReference)
		**out = **in
	}
	if in.CABundle != nil {
		in, out := &in.CABundle, &out.CABundle
		*out = make([]byte, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClientConfig.
func (in *ClientConfig) DeepCopy() *ClientConfig {
	if in == nil {
		return nil
	}
	out := new(ClientConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ObjectReference) DeepCopyInto(out *ObjectReference) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ObjectReference.
func (in *ObjectReference) DeepCopy() *ObjectReference {
	if in == nil {
		return nil
	}
	out := new(ObjectReference)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Param) DeepCopyInto(out *Param) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Param.
func (in *Param) DeepCopy() *Param {
	if in == nil {
		return nil
	}
	out := new(Param)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RemoveKeyTransform) DeepCopyInto(out *RemoveKeyTransform) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RemoveKeyTransform.
func (in *RemoveKeyTransform) DeepCopy() *RemoveKeyTransform {
	if in == nil {
		return nil
	}
	out := new(RemoveKeyTransform)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RenameKeyTransform) DeepCopyInto(out *RenameKeyTransform) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RenameKeyTransform.
func (in *RenameKeyTransform) DeepCopy() *RenameKeyTransform {
	if in == nil {
		return nil
	}
	out := new(RenameKeyTransform)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SecretTransform) DeepCopyInto(out *SecretTransform) {
	*out = *in
	if in.RenameKey != nil {
		in, out := &in.RenameKey, &out.RenameKey
		*out = new(RenameKeyTransform)
		**out = **in
	}
	if in.AddKey != nil {
		in, out := &in.AddKey, &out.AddKey
		*out = new(AddKeyTransform)
		(*in).DeepCopyInto(*out)
	}
	if in.AddKeysFrom != nil {
		in, out := &in.AddKeysFrom, &out.AddKeysFrom
		*out = new(AddKeysFromTransform)
		(*in).DeepCopyInto(*out)
	}
	if in.RemoveKey != nil {
		in, out := &in.RemoveKey, &out.RemoveKey
		*out = new(RemoveKeyTransform)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SecretTransform.
func (in *SecretTransform) DeepCopy() *SecretTransform {
	if in == nil {
		return nil
	}
	out := new(SecretTransform)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ServiceReference) DeepCopyInto(out *ServiceReference) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ServiceReference.
func (in *ServiceReference) DeepCopy() *ServiceReference {
	if in == nil {
		return nil
	}
	out := new(ServiceReference)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *StashAddon) DeepCopyInto(out *StashAddon) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.Stash.DeepCopyInto(&out.Stash)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new StashAddon.
func (in *StashAddon) DeepCopy() *StashAddon {
	if in == nil {
		return nil
	}
	out := new(StashAddon)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *StashAddon) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *StashAddonSpec) DeepCopyInto(out *StashAddonSpec) {
	*out = *in
	in.Addon.DeepCopyInto(&out.Addon)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new StashAddonSpec.
func (in *StashAddonSpec) DeepCopy() *StashAddonSpec {
	if in == nil {
		return nil
	}
	out := new(StashAddonSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *StashTaskSpec) DeepCopyInto(out *StashTaskSpec) {
	*out = *in
	in.BackupTask.DeepCopyInto(&out.BackupTask)
	in.RestoreTask.DeepCopyInto(&out.RestoreTask)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new StashTaskSpec.
func (in *StashTaskSpec) DeepCopy() *StashTaskSpec {
	if in == nil {
		return nil
	}
	out := new(StashTaskSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TaskRef) DeepCopyInto(out *TaskRef) {
	*out = *in
	if in.Params != nil {
		in, out := &in.Params, &out.Params
		*out = make([]Param, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TaskRef.
func (in *TaskRef) DeepCopy() *TaskRef {
	if in == nil {
		return nil
	}
	out := new(TaskRef)
	in.DeepCopyInto(out)
	return out
}
