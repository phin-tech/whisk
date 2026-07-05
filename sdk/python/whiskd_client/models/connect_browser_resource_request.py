from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="ConnectBrowserResourceRequest")


@_attrs_define
class ConnectBrowserResourceRequest:
    """
    Attributes:
        acknowledge_browser_control_risk (bool):
        cdp_url (str):
        name (str | Unset):
    """

    acknowledge_browser_control_risk: bool
    cdp_url: str
    name: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        acknowledge_browser_control_risk = self.acknowledge_browser_control_risk

        cdp_url = self.cdp_url

        name = self.name

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "acknowledgeBrowserControlRisk": acknowledge_browser_control_risk,
                "cdpUrl": cdp_url,
            }
        )
        if name is not UNSET:
            field_dict["name"] = name

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        acknowledge_browser_control_risk = d.pop("acknowledgeBrowserControlRisk")

        cdp_url = d.pop("cdpUrl")

        name = d.pop("name", UNSET)

        connect_browser_resource_request = cls(
            acknowledge_browser_control_risk=acknowledge_browser_control_risk,
            cdp_url=cdp_url,
            name=name,
        )

        connect_browser_resource_request.additional_properties = d
        return connect_browser_resource_request

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
