#ifndef T_PATH_RULES_H
#define T_PATH_RULES_H

#include <stdlib.h>
#include <string>
#include <utility>
#include <vector>

#include "thrift/platform.h"

/**
 * Rules keyed by '/'-separated prefix. A rule applies to a path equal to or
 * under its prefix, and the rule with the longest prefix wins.
 */
class t_path_rules {
public:
  void add(const std::string& prefix, const std::string& value) {
    rules_.push_back({prefix, value});
  }

  /**
   * Adds a rule on a filesystem path, resolved with realpath when it exists
   * so that it compares with the resolved paths of programs.
   */
  void add_path(const std::string& path, const std::string& value) {
    char rp[THRIFT_PATH_MAX];

    add(realpath(path.c_str(), rp) != NULL ? rp : path, value);
  }

  /**
   * Returns the value of the rule matching path, or fallback.
   */
  std::string lookup(const std::string& path, const std::string& fallback) const {
    const std::pair<std::string, std::string>* rule = match(path);

    return rule != NULL ? rule->second : fallback;
  }

  /**
   * Returns path with the prefix of the matching rule replaced by its value.
   */
  std::string rewrite(const std::string& path) const {
    const std::pair<std::string, std::string>* rule = match(path);

    if (rule == NULL) {
      return path;
    }

    return rule->second + path.substr(rule->first.size());
  }

private:
  std::vector<std::pair<std::string, std::string>> rules_;

  const std::pair<std::string, std::string>* match(const std::string& path) const {
    const std::pair<std::string, std::string>* best = NULL;

    for (const auto& rule : rules_) {
      if (path != rule.first && path.rfind(rule.first + "/", 0) != 0) {
        continue;
      }

      if (best == NULL || rule.first.size() > best->first.size()) {
        best = &rule;
      }
    }

    return best;
  }
};

#endif
