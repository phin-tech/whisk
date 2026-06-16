from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="AgentHookIntegration")


@_attrs_define
class AgentHookIntegration:
    """
    Attributes:
        config_path (str):
        helper_path (str):
        latest_version (str):
        manifest_path (str):
        provider (str):
        status (str):
        detail (str | Unset):
        installed_version (str | Unset):
    """

    config_path: str
    helper_path: str
    latest_version: str
    manifest_path: str
    provider: str
    status: str
    detail: str | Unset = UNSET
    installed_version: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        config_path = self.config_path

        helper_path = self.helper_path

        latest_version = self.latest_version

        manifest_path = self.manifest_path

        provider = self.provider

        status = self.status

        detail = self.detail

        installed_version = self.installed_version

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "configPath": config_path,
                "helperPath": helper_path,
                "latestVersion": latest_version,
                "manifestPath": manifest_path,
                "provider": provider,
                "status": status,
            }
        )
        if detail is not UNSET:
            field_dict["detail"] = detail
        if installed_version is not UNSET:
            field_dict["installedVersion"] = installed_version

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        config_path = d.pop("configPath")

        helper_path = d.pop("helperPath")

        latest_version = d.pop("latestVersion")

        manifest_path = d.pop("manifestPath")

        provider = d.pop("provider")

        status = d.pop("status")

        detail = d.pop("detail", UNSET)

        installed_version = d.pop("installedVersion", UNSET)

        agent_hook_integration = cls(
            config_path=config_path,
            helper_path=helper_path,
            latest_version=latest_version,
            manifest_path=manifest_path,
            provider=provider,
            status=status,
            detail=detail,
            installed_version=installed_version,
        )

        agent_hook_integration.additional_properties = d
        return agent_hook_integration

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
