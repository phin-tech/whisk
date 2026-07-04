from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="PluginPermissions")


@_attrs_define
class PluginPermissions:
    """
    Attributes:
        env_prefixes (list[str] | Unset):
        network (list[str] | Unset):
        pty_output (bool | Unset):
    """

    env_prefixes: list[str] | Unset = UNSET
    network: list[str] | Unset = UNSET
    pty_output: bool | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        env_prefixes: list[str] | Unset = UNSET
        if not isinstance(self.env_prefixes, Unset):
            env_prefixes = self.env_prefixes

        network: list[str] | Unset = UNSET
        if not isinstance(self.network, Unset):
            network = self.network

        pty_output = self.pty_output

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if env_prefixes is not UNSET:
            field_dict["envPrefixes"] = env_prefixes
        if network is not UNSET:
            field_dict["network"] = network
        if pty_output is not UNSET:
            field_dict["ptyOutput"] = pty_output

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        env_prefixes = cast(list[str], d.pop("envPrefixes", UNSET))

        network = cast(list[str], d.pop("network", UNSET))

        pty_output = d.pop("ptyOutput", UNSET)

        plugin_permissions = cls(
            env_prefixes=env_prefixes,
            network=network,
            pty_output=pty_output,
        )

        plugin_permissions.additional_properties = d
        return plugin_permissions

    @property
    def additional_keys(self) -> list[str]:
        return list(self.additional_properties.keys())

    def __getitem__(self, key: str) -> Any:
        return self.additional_properties[key]

    def __setitem__(self, key: str, value: Any) -> None:
        self.additional_properties[key] = value

    def __delitem__(self, key: str) -> None:
        del self.additional_properties[key]

    def __contains__(self, key: str) -> bool:
        return key in self.additional_properties
