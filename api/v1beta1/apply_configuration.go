package v1beta1

import (
	"encoding/json"

	corev1apply "k8s.io/client-go/applyconfigurations/core/v1"
)

type EnvVarApplyConfigurationList []*corev1apply.EnvVarApplyConfiguration

func (c *EnvVarApplyConfigurationList) DeepCopy() *EnvVarApplyConfigurationList {
	out := new(EnvVarApplyConfigurationList)
	bytes, err := json.Marshal(c)
	if err != nil {
		panic("Failed to marshal")
	}
	if err := json.Unmarshal(bytes, out); err != nil {
		panic("Failed to unmarshal")
	}
	return out
}

func (l EnvVarApplyConfigurationList) Ref() []*corev1apply.EnvVarApplyConfiguration {
	if l == nil {
		return nil
	}
	s := make([]*corev1apply.EnvVarApplyConfiguration, len(l))
	copy(s, l)
	return s
}

type AffinityApplyConfiguration corev1apply.AffinityApplyConfiguration

func (c *AffinityApplyConfiguration) DeepCopy() *AffinityApplyConfiguration {
	out := new(AffinityApplyConfiguration)
	bytes, err := json.Marshal(c)
	if err != nil {
		panic("Failed to marshal")
	}
	if err := json.Unmarshal(bytes, out); err != nil {
		panic("Failed to unmarshal")
	}
	return out
}

func (c *AffinityApplyConfiguration) Ref() *corev1apply.AffinityApplyConfiguration {
	return (*corev1apply.AffinityApplyConfiguration)(c)
}

type TolerationApplyConfigurationList []*corev1apply.TolerationApplyConfiguration

func (l *TolerationApplyConfigurationList) DeepCopy() *TolerationApplyConfigurationList {
	out := new(TolerationApplyConfigurationList)
	bytes, err := json.Marshal(l)
	if err != nil {
		panic("Failed to marshal")
	}
	if err := json.Unmarshal(bytes, out); err != nil {
		panic("Failed to unmarshal")
	}
	return out
}

func (l TolerationApplyConfigurationList) Ref() []*corev1apply.TolerationApplyConfiguration {
	if l == nil {
		return nil
	}
	s := make([]*corev1apply.TolerationApplyConfiguration, len(l))
	copy(s, l)
	return s
}
