from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

if TYPE_CHECKING:
    from ..models.skill import Skill
    from ..models.skill_source import SkillSource


T = TypeVar("T", bound="SkillCatalog")


@_attrs_define
class SkillCatalog:
    """
    Attributes:
        scanned_at (datetime.datetime):
        skills (list[Skill] | None):
        sources (list[SkillSource] | None):
    """

    scanned_at: datetime.datetime
    skills: list[Skill] | None
    sources: list[SkillSource] | None
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        scanned_at = self.scanned_at.isoformat()

        skills: list[dict[str, Any]] | None
        if isinstance(self.skills, list):
            skills = []
            for skills_type_0_item_data in self.skills:
                skills_type_0_item = skills_type_0_item_data.to_dict()
                skills.append(skills_type_0_item)

        else:
            skills = self.skills

        sources: list[dict[str, Any]] | None
        if isinstance(self.sources, list):
            sources = []
            for sources_type_0_item_data in self.sources:
                sources_type_0_item = sources_type_0_item_data.to_dict()
                sources.append(sources_type_0_item)

        else:
            sources = self.sources

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "scannedAt": scanned_at,
                "skills": skills,
                "sources": sources,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.skill import Skill
        from ..models.skill_source import SkillSource

        d = dict(src_dict)
        scanned_at = datetime.datetime.fromisoformat(d.pop("scannedAt"))

        def _parse_skills(data: object) -> list[Skill] | None:
            if data is None:
                return data
            try:
                if not isinstance(data, list):
                    raise TypeError()
                skills_type_0 = []
                _skills_type_0 = data
                for skills_type_0_item_data in _skills_type_0:
                    skills_type_0_item = Skill.from_dict(skills_type_0_item_data)

                    skills_type_0.append(skills_type_0_item)

                return skills_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(list[Skill] | None, data)

        skills = _parse_skills(d.pop("skills"))

        def _parse_sources(data: object) -> list[SkillSource] | None:
            if data is None:
                return data
            try:
                if not isinstance(data, list):
                    raise TypeError()
                sources_type_0 = []
                _sources_type_0 = data
                for sources_type_0_item_data in _sources_type_0:
                    sources_type_0_item = SkillSource.from_dict(
                        sources_type_0_item_data
                    )

                    sources_type_0.append(sources_type_0_item)

                return sources_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(list[SkillSource] | None, data)

        sources = _parse_sources(d.pop("sources"))

        skill_catalog = cls(
            scanned_at=scanned_at,
            skills=skills,
            sources=sources,
        )

        skill_catalog.additional_properties = d
        return skill_catalog

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
