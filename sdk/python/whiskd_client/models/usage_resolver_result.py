from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.usage_resolver_metric import UsageResolverMetric
    from ..models.usage_resolver_result_meta import UsageResolverResultMeta


T = TypeVar("T", bound="UsageResolverResult")


@_attrs_define
class UsageResolverResult:
    """
    Attributes:
        metrics (list[UsageResolverMetric] | None):
        fetched_at (datetime.datetime | None | Unset):
        meta (UsageResolverResultMeta | Unset):
        summary (str | Unset):
    """

    metrics: list[UsageResolverMetric] | None
    fetched_at: datetime.datetime | None | Unset = UNSET
    meta: UsageResolverResultMeta | Unset = UNSET
    summary: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        metrics: list[dict[str, Any]] | None
        if isinstance(self.metrics, list):
            metrics = []
            for metrics_type_0_item_data in self.metrics:
                metrics_type_0_item = metrics_type_0_item_data.to_dict()
                metrics.append(metrics_type_0_item)

        else:
            metrics = self.metrics

        fetched_at: None | str | Unset
        if isinstance(self.fetched_at, Unset):
            fetched_at = UNSET
        elif isinstance(self.fetched_at, datetime.datetime):
            fetched_at = self.fetched_at.isoformat()
        else:
            fetched_at = self.fetched_at

        meta: dict[str, Any] | Unset = UNSET
        if not isinstance(self.meta, Unset):
            meta = self.meta.to_dict()

        summary = self.summary

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "metrics": metrics,
            }
        )
        if fetched_at is not UNSET:
            field_dict["fetchedAt"] = fetched_at
        if meta is not UNSET:
            field_dict["meta"] = meta
        if summary is not UNSET:
            field_dict["summary"] = summary

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.usage_resolver_metric import UsageResolverMetric
        from ..models.usage_resolver_result_meta import UsageResolverResultMeta

        d = dict(src_dict)

        def _parse_metrics(data: object) -> list[UsageResolverMetric] | None:
            if data is None:
                return data
            try:
                if not isinstance(data, list):
                    raise TypeError()
                metrics_type_0 = []
                _metrics_type_0 = data
                for metrics_type_0_item_data in _metrics_type_0:
                    metrics_type_0_item = UsageResolverMetric.from_dict(
                        metrics_type_0_item_data
                    )

                    metrics_type_0.append(metrics_type_0_item)

                return metrics_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(list[UsageResolverMetric] | None, data)

        metrics = _parse_metrics(d.pop("metrics"))

        def _parse_fetched_at(data: object) -> datetime.datetime | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                fetched_at_type_0 = datetime.datetime.fromisoformat(data)

                return fetched_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None | Unset, data)

        fetched_at = _parse_fetched_at(d.pop("fetchedAt", UNSET))

        _meta = d.pop("meta", UNSET)
        meta: UsageResolverResultMeta | Unset
        if isinstance(_meta, Unset):
            meta = UNSET
        else:
            meta = UsageResolverResultMeta.from_dict(_meta)

        summary = d.pop("summary", UNSET)

        usage_resolver_result = cls(
            metrics=metrics,
            fetched_at=fetched_at,
            meta=meta,
            summary=summary,
        )

        usage_resolver_result.additional_properties = d
        return usage_resolver_result

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
