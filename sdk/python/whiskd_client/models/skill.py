from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="Skill")


@_attrs_define
class Skill:
    """
    Attributes:
        directory_path (str):
        file_count (int):
        id (str):
        name (str):
        providers (list[str] | None):
        root_path (str):
        skill_file_path (str):
        source_kind (str):
        source_label (str):
        updated_at (datetime.datetime):
        description (str | Unset):
    """

    directory_path: str
    file_count: int
    id: str
    name: str
    providers: list[str] | None
    root_path: str
    skill_file_path: str
    source_kind: str
    source_label: str
    updated_at: datetime.datetime
    description: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        directory_path = self.directory_path

        file_count = self.file_count

        id = self.id

        name = self.name

        providers: list[str] | None
        if isinstance(self.providers, list):
            providers = self.providers

        else:
            providers = self.providers

        root_path = self.root_path

        skill_file_path = self.skill_file_path

        source_kind = self.source_kind

        source_label = self.source_label

        updated_at = self.updated_at.isoformat()

        description = self.description

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "directoryPath": directory_path,
                "fileCount": file_count,
                "id": id,
                "name": name,
                "providers": providers,
                "rootPath": root_path,
                "skillFilePath": skill_file_path,
                "sourceKind": source_kind,
                "sourceLabel": source_label,
                "updatedAt": updated_at,
            }
        )
        if description is not UNSET:
            field_dict["description"] = description

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        directory_path = d.pop("directoryPath")

        file_count = d.pop("fileCount")

        id = d.pop("id")

        name = d.pop("name")

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

        root_path = d.pop("rootPath")

        skill_file_path = d.pop("skillFilePath")

        source_kind = d.pop("sourceKind")

        source_label = d.pop("sourceLabel")

        updated_at = datetime.datetime.fromisoformat(d.pop("updatedAt"))

        description = d.pop("description", UNSET)

        skill = cls(
            directory_path=directory_path,
            file_count=file_count,
            id=id,
            name=name,
            providers=providers,
            root_path=root_path,
            skill_file_path=skill_file_path,
            source_kind=source_kind,
            source_label=source_label,
            updated_at=updated_at,
            description=description,
        )

        skill.additional_properties = d
        return skill

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
