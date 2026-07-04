from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="UsageResolverMetric")


@_attrs_define
class UsageResolverMetric:
    """
    Attributes:
        id (str):
        kind (str):
        label (str | Unset):
        limit (float | None | Unset):
        remaining (float | None | Unset):
        reset_at (datetime.datetime | None | Unset):
        unit (str | Unset):
        used (float | None | Unset):
    """

    id: str
    kind: str
    label: str | Unset = UNSET
    limit: float | None | Unset = UNSET
    remaining: float | None | Unset = UNSET
    reset_at: datetime.datetime | None | Unset = UNSET
    unit: str | Unset = UNSET
    used: float | None | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        id = self.id

        kind = self.kind

        label = self.label

        limit: float | None | Unset
        if isinstance(self.limit, Unset):
            limit = UNSET
        else:
            limit = self.limit

        remaining: float | None | Unset
        if isinstance(self.remaining, Unset):
            remaining = UNSET
        else:
            remaining = self.remaining

        reset_at: None | str | Unset
        if isinstance(self.reset_at, Unset):
            reset_at = UNSET
        elif isinstance(self.reset_at, datetime.datetime):
            reset_at = self.reset_at.isoformat()
        else:
            reset_at = self.reset_at

        unit = self.unit

        used: float | None | Unset
        if isinstance(self.used, Unset):
            used = UNSET
        else:
            used = self.used

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "id": id,
                "kind": kind,
            }
        )
        if label is not UNSET:
            field_dict["label"] = label
        if limit is not UNSET:
            field_dict["limit"] = limit
        if remaining is not UNSET:
            field_dict["remaining"] = remaining
        if reset_at is not UNSET:
            field_dict["resetAt"] = reset_at
        if unit is not UNSET:
            field_dict["unit"] = unit
        if used is not UNSET:
            field_dict["used"] = used

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        kind = d.pop("kind")

        label = d.pop("label", UNSET)

        def _parse_limit(data: object) -> float | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(float | None | Unset, data)

        limit = _parse_limit(d.pop("limit", UNSET))

        def _parse_remaining(data: object) -> float | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(float | None | Unset, data)

        remaining = _parse_remaining(d.pop("remaining", UNSET))

        def _parse_reset_at(data: object) -> datetime.datetime | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                reset_at_type_0 = datetime.datetime.fromisoformat(data)

                return reset_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None | Unset, data)

        reset_at = _parse_reset_at(d.pop("resetAt", UNSET))

        unit = d.pop("unit", UNSET)

        def _parse_used(data: object) -> float | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(float | None | Unset, data)

        used = _parse_used(d.pop("used", UNSET))

        usage_resolver_metric = cls(
            id=id,
            kind=kind,
            label=label,
            limit=limit,
            remaining=remaining,
            reset_at=reset_at,
            unit=unit,
            used=used,
        )

        usage_resolver_metric.additional_properties = d
        return usage_resolver_metric

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
