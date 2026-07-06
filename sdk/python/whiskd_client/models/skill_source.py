from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="SkillSource")


@_attrs_define
class SkillSource:
    """
    Attributes:
        exists (bool):
        id (str):
        kind (str):
        label (str):
        path (str):
        providers (list[str] | None):
        skipped_reason (str | Unset):
    """

    exists: bool
    id: str
    kind: str
    label: str
    path: str
    providers: list[str] | None
    skipped_reason: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        exists = self.exists

        id = self.id

        kind = self.kind

        label = self.label

        path = self.path

        providers: list[str] | None
        if isinstance(self.providers, list):
            providers = self.providers

        else:
            providers = self.providers

        skipped_reason = self.skipped_reason

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "exists": exists,
                "id": id,
                "kind": kind,
                "label": label,
                "path": path,
                "providers": providers,
            }
        )
        if skipped_reason is not UNSET:
            field_dict["skippedReason"] = skipped_reason

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        exists = d.pop("exists")

        id = d.pop("id")

        kind = d.pop("kind")

        label = d.pop("label")

        path = d.pop("path")

        def _parse_providers(data: object) -> list[str] | None:
            if data is None:
                return data
            try:
                if not isinstance(data, list):
                    raise TypeError()
                providers_type_0 = cast(list[str], data)

                return providers_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(list[str] | None, data)

        providers = _parse_providers(d.pop("providers"))

        skipped_reason = d.pop("skippedReason", UNSET)

        skill_source = cls(
            exists=exists,
            id=id,
            kind=kind,
            label=label,
            path=path,
            providers=providers,
            skipped_reason=skipped_reason,
        )

        skill_source.additional_properties = d
        return skill_source

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
