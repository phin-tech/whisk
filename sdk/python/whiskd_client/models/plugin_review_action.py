from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="PluginReviewAction")


@_attrs_define
class PluginReviewAction:
    """
    Attributes:
        id (str):
        label (str):
        blocking (bool | Unset):
        has_submit (bool | Unset):
        output_cap_bytes (int | Unset):
        scope (str | Unset):
        timeout_ms (int | Unset):
        url_template (str | Unset):
    """

    id: str
    label: str
    blocking: bool | Unset = UNSET
    has_submit: bool | Unset = UNSET
    output_cap_bytes: int | Unset = UNSET
    scope: str | Unset = UNSET
    timeout_ms: int | Unset = UNSET
    url_template: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        id = self.id

        label = self.label

        blocking = self.blocking

        has_submit = self.has_submit

        output_cap_bytes = self.output_cap_bytes

        scope = self.scope

        timeout_ms = self.timeout_ms

        url_template = self.url_template

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "id": id,
                "label": label,
            }
        )
        if blocking is not UNSET:
            field_dict["blocking"] = blocking
        if has_submit is not UNSET:
            field_dict["hasSubmit"] = has_submit
        if output_cap_bytes is not UNSET:
            field_dict["outputCapBytes"] = output_cap_bytes
        if scope is not UNSET:
            field_dict["scope"] = scope
        if timeout_ms is not UNSET:
            field_dict["timeoutMs"] = timeout_ms
        if url_template is not UNSET:
            field_dict["urlTemplate"] = url_template

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        label = d.pop("label")

        blocking = d.pop("blocking", UNSET)

        has_submit = d.pop("hasSubmit", UNSET)

        output_cap_bytes = d.pop("outputCapBytes", UNSET)

        scope = d.pop("scope", UNSET)

        timeout_ms = d.pop("timeoutMs", UNSET)

        url_template = d.pop("urlTemplate", UNSET)

        plugin_review_action = cls(
            id=id,
            label=label,
            blocking=blocking,
            has_submit=has_submit,
            output_cap_bytes=output_cap_bytes,
            scope=scope,
            timeout_ms=timeout_ms,
            url_template=url_template,
        )

        plugin_review_action.additional_properties = d
        return plugin_review_action

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
