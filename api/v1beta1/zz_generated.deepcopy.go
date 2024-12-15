//go:build !ignore_autogenerated

// Code generated by controller-gen. DO NOT EDIT.

package v1beta1

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CloudflareTunnel) DeepCopyInto(out *CloudflareTunnel) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CloudflareTunnel.
func (in *CloudflareTunnel) DeepCopy() *CloudflareTunnel {
	if in == nil {
		return nil
	}
	out := new(CloudflareTunnel)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *CloudflareTunnel) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CloudflareTunnelList) DeepCopyInto(out *CloudflareTunnelList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]CloudflareTunnel, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CloudflareTunnelList.
func (in *CloudflareTunnelList) DeepCopy() *CloudflareTunnelList {
	if in == nil {
		return nil
	}
	out := new(CloudflareTunnelList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *CloudflareTunnelList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CloudflareTunnelSecret) DeepCopyInto(out *CloudflareTunnelSecret) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CloudflareTunnelSecret.
func (in *CloudflareTunnelSecret) DeepCopy() *CloudflareTunnelSecret {
	if in == nil {
		return nil
	}
	out := new(CloudflareTunnelSecret)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CloudflareTunnelSpec) DeepCopyInto(out *CloudflareTunnelSpec) {
	*out = *in
	out.Secret = in.Secret
	if in.CAPool != nil {
		in, out := &in.CAPool, &out.CAPool
		*out = new(string)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CloudflareTunnelSpec.
func (in *CloudflareTunnelSpec) DeepCopy() *CloudflareTunnelSpec {
	if in == nil {
		return nil
	}
	out := new(CloudflareTunnelSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CloudflareTunnelStatus) DeepCopyInto(out *CloudflareTunnelStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]v1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CloudflareTunnelStatus.
func (in *CloudflareTunnelStatus) DeepCopy() *CloudflareTunnelStatus {
	if in == nil {
		return nil
	}
	out := new(CloudflareTunnelStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterCloudflareTunnel) DeepCopyInto(out *ClusterCloudflareTunnel) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterCloudflareTunnel.
func (in *ClusterCloudflareTunnel) DeepCopy() *ClusterCloudflareTunnel {
	if in == nil {
		return nil
	}
	out := new(ClusterCloudflareTunnel)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ClusterCloudflareTunnel) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterCloudflareTunnelList) DeepCopyInto(out *ClusterCloudflareTunnelList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ClusterCloudflareTunnel, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterCloudflareTunnelList.
func (in *ClusterCloudflareTunnelList) DeepCopy() *ClusterCloudflareTunnelList {
	if in == nil {
		return nil
	}
	out := new(ClusterCloudflareTunnelList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ClusterCloudflareTunnelList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}
