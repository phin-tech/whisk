from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="Item")


@_attrs_define
class Item:
    """
    Attributes:
        id (str):
        kind (str):
        label (str):
        selected_by_default (bool):
        status (str):
        target (str):
        description (str | Unset):
        detail (str | Unset):
        hash_ (str | Unset):
        installed_hash (str | Unset):
        installed_version (str | Unset):
        latest_version (str | Unset):
        path (str | Unset):
        version (str | Unset):
    """

    id: str
    kind: str
    label: str
    selected_by_default: bool
    status: str
    target: str
    description: str | Unset = UNSET
    detail: str | Unset = UNSET
    hash_: str | Unset = UNSET
    installed_hash: str | Unset = UNSET
    installed_version: str | Unset = UNSET
    latest_version: str | Unset = UNSET
    path: str | Unset = UNSET
    version: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        id = self.id

        kind = self.kind

        label = self.label

        selected_by_default = self.selected_by_default

        status = self.status

        target = self.target

        description = self.description

        detail = self.detail

        hash_ = self.hash_

        installed_hash = self.installed_hash

        installed_version = self.installed_version

        latest_version = self.latest_version

        path = self.path

        version = self.version

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "id": id,
                "kind": kind,
                "label": label,
                "selectedByDefault": selected_by_default,
                "status": status,
                "target": target,
            }
        )
        if description is not UNSET:
            field_dict["description"] = description
        if detail is not UNSET:
            field_dict["detail"] = detail
        if hash_ is not UNSET:
            field_dict["hash"] = hash_
        if installed_hash is not UNSET:
            field_dict["installedHash"] = installed_hash
        if installed_version is not UNSET:
            field_dict["installedVersion"] = installed_version
        if latest_version is not UNSET:
            field_dict["latestVersion"] = latest_version
        if path is not UNSET:
            field_dict["path"] = path
        if version is not UNSET:
            field_dict["version"] = version

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        kind = d.pop("kind")

        label = d.pop("label")

        selected_by_default = d.pop("selectedByDefault")

        status = d.pop("status")

        target = d.pop("target")

        description = d.pop("description", UNSET)

        detail = d.pop("detail", UNSET)

        hash_ = d.pop("hash", UNSET)

        installed_hash = d.pop("installedHash", UNSET)

        installed_version = d.pop("installedVersion", UNSET)

        latest_version = d.pop("latestVersion", UNSET)

        path = d.pop("path", UNSET)

        version = d.pop("version", UNSET)

        item = cls(
            id=id,
            kind=kind,
            label=label,
            selected_by_default=selected_by_default,
            status=status,
            target=target,
            description=description,
            detail=detail,
            hash_=hash_,
            installed_hash=installed_hash,
            installed_version=installed_version,
            latest_version=latest_version,
            path=path,
            version=version,
        )

        item.additional_properties = d
        return item

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
